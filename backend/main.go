package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"clinic-system/backend/models"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type ParseNoteRequest struct {
	PatientID   uint    `json:"patient_id" binding:"required"`
	RawNote     string  `json:"raw_note" binding:"required"`
	ExtraCharge float64 `json:"extra_charge"`
}

type ParseNoteResponse struct {
	RecordID    uint               `json:"record_id"`
	PatientID   uint               `json:"patient_id"`
	TotalBill   float64            `json:"total_bill"`
	ExtraCharge float64            `json:"extra_charge"`
	ParsedItems []models.ParsedItem `json:"parsed_items"`
	Notes       string             `json:"notes"`
}

type AIParseRequest struct {
	PatientID   uint    `json:"patient_id" binding:"required"`
	RawNote     string  `json:"raw_note" binding:"required"`
	ExtraCharge float64 `json:"extra_charge"`
}

type AIParseResult struct {
	Drugs        []models.ParsedItem `json:"drugs"`
	LabTests     []models.ParsedItem `json:"lab_tests"`
	Observations []models.ParsedItem `json:"observations"`
	TotalBill    float64             `json:"total_bill"`
	ExtraCharge  float64             `json:"extra_charge"`
	Note         string              `json:"note"`
}

func main() {
	// 1. Connect to Database
	ConnectDatabase()

	// 2. Seed fallback prices (if missing)
	seedDefaultPriceList()

	// 3. Initialize Router
	r := gin.Default()
	r.Use(cors.Default())

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "Clinic API running"})
	})

	r.GET("/test-db", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "Database connection is successful!"})
	})

	r.POST("/parse-note", parseAndSaveNote)
	r.POST("/ai-parse", aiParseNote)
	r.GET("/billing/:patient_id", getBilling)
	r.POST("/patients", createPatient)
	r.GET("/patients", getPatients)
	r.PUT("/patients/:id", updatePatient)
	r.DELETE("/patients/:id", deletePatient)
	r.PUT("/medical-records/:id", updateMedicalRecord)
	r.DELETE("/medical-records/:id", deleteMedicalRecord)

	// Keep standard 8080 for backend endpoints
	r.Run(":8080")
}

func parseAndSaveNote(c *gin.Context) {
	var req ParseNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Save system raw note first
	record := models.MedicalRecord{
		PatientID: req.PatientID,
		RawNote:   req.RawNote,
	}
	if err := DB.Create(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create medical record"})
		return
	}

	// Save extra charges before parsing
	record.ExtraCharge = req.ExtraCharge

	// Prepare AI-like classification using price list heuristics
	priceList := []models.PriceList{}
	if err := DB.Find(&priceList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load price list"})
		return
	}

	lowerRaw := strings.ToLower(req.RawNote)
	parsed := []models.ParsedItem{}
	seenItems := map[string]bool{}

	for _, p := range priceList {
		if strings.Contains(lowerRaw, strings.ToLower(p.ItemName)) {
			if seenItems[p.ItemName] {
				continue
			}
			dosage := ""
			if strings.EqualFold(p.Category, "Drug") {
				dosage = extractDosage(req.RawNote)
			}
			item := models.ParsedItem{
				RecordID:   record.ID,
				Category:   p.Category,
				ItemName:   p.ItemName,
				Dosage:     dosage,
				Price:      p.Price,
				Confidence: 1.0,
			}
			parsed = append(parsed, item)
			seenItems[p.ItemName] = true
		}
	}

	// Add a fallback observation entry for remaining text if no parsed items
	if len(parsed) == 0 {
		lower := strings.ToLower(req.RawNote)
		confidence := 0.3
		if strings.Contains(lower, "pain") || strings.Contains(lower, "cough") || strings.Contains(lower, "fever") {
			confidence = 0.75
		}
		parsed = append(parsed, models.ParsedItem{
			RecordID:   record.ID,
			Category:   "Observation",
			ItemName:   "Clinical Notes",
			Dosage:     "",
			Price:      0,
			Confidence: confidence,
		})
	}

	// Insert parsed items
	for i := range parsed {
		if err := DB.Create(&parsed[i]).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save parsed item"})
			return
		}
	}

	// Calculate total and update medical record
	total := 0.0
	for _, item := range parsed {
		if item.Confidence == 0 {
			item.Confidence = 1.0
		}
		total += item.Price
	}
	// Include manual extra charge
	total += record.ExtraCharge

	// Use explicit Update instead of Save to ensure fields persist
	if err := DB.Model(&record).Updates(models.MedicalRecord{
		ExtraCharge: record.ExtraCharge,
		TotalBill:   total,
	}).Error; err != nil {
		log.Printf("Failed to update record: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update total bill: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, ParseNoteResponse{
		RecordID:    record.ID,
		PatientID:   req.PatientID,
		TotalBill:   total,
		ExtraCharge: req.ExtraCharge,
		ParsedItems: parsed,
		Notes:       req.RawNote,
	})
}

func aiParseNote(c *gin.Context) {
	var req AIParseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items, err := callOpenAIClassification(req.RawNote)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "AI parse request failed: " + err.Error()})
		return
	}

	// merge result, assign price from price list and patient record etc
	priceList := []models.PriceList{}
	_ = DB.Find(&priceList)

	drugItems := []models.ParsedItem{}
	labItems := []models.ParsedItem{}
	obsItems := []models.ParsedItem{}
	calc := func(pi models.ParsedItem) float64 {
		for _, p := range priceList {
			if strings.EqualFold(p.ItemName, pi.ItemName) {
				pi.Price = p.Price
				return p.Price
			}
		}
		return 0
	}

	total := 0.0
	for _, item := range items {
		price := calc(item)
		item.Price = price
		total += price
		if strings.EqualFold(item.Category, "Drug") {
			drugItems = append(drugItems, item)
		} else if strings.EqualFold(item.Category, "Lab Test") {
			labItems = append(labItems, item)
		} else {
			obsItems = append(obsItems, item)
		}
	}

	total += req.ExtraCharge

	c.JSON(http.StatusOK, AIParseResult{
		Drugs:        drugItems,
		LabTests:     labItems,
		Observations: obsItems,
		TotalBill:    total,
		ExtraCharge:  req.ExtraCharge,
		Note:         req.RawNote,
	})
}

func callOpenAIClassification(rawNote string) ([]models.ParsedItem, error) {
	if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
		return callGeminiClassification(rawNote, geminiKey)
	}

	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		return callOpenAIChatClassification(rawNote, openaiKey)
	}

	return nil, errors.New("GEMINI_API_KEY or OPENAI_API_KEY must be set")
}

func callOpenAIChatClassification(rawNote, apiKey string) ([]models.ParsedItem, error) {
	prompt := `You are a medical assistant. Parse this clinic note into JSON with keys drugs, lab_tests, observations. Each item should include item_name, dosage (if present), and category. Return strict JSON only.`
	message := []map[string]string{
		{"role": "system", "content": prompt},
		{"role": "user", "content": rawNote},
	}

	payload := map[string]interface{}{
		"model": "gpt-4.1-mini",
		"messages": message,
		"temperature": 0.1,
	}

	bodyb, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyb))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return parseAIResponse(req)
}

func callGeminiClassification(rawNote, apiKey string) ([]models.ParsedItem, error) {
	prompt := `You are a medical assistant. Parse this clinic note into JSON with keys drugs, lab_tests, observations. Each item should include item_name, dosage (if present), and category. Return strict JSON only.`
	messages := []map[string]string{
		{"role": "system", "content": prompt},
		{"role": "user", "content": rawNote},
	}

	payload := map[string]interface{}{
		"prompt": map[string]interface{}{
			"messages": messages,
		},
		"temperature": 0.1,
		"maxOutputTokens": 512,
	}

	endpoint := fmt.Sprintf("https://generativeai.googleapis.com/v1beta2/models/gemini-1.0:generate?key=%s", apiKey)
	bodyb, _ := json.Marshal(payload)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(bodyb))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("gemini status %d: %s", resp.StatusCode, string(b))
	}

	var gemResp struct {
		Candidates []struct {
			Content string `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gemResp); err != nil {
		return nil, err
	}
	if len(gemResp.Candidates) == 0 {
		return nil, errors.New("no candidates in gemini response")
	}

	text := gemResp.Candidates[0].Content
	parsed := struct {
		Drugs        []models.ParsedItem `json:"drugs"`
		LabTests     []models.ParsedItem `json:"lab_tests"`
		Observations []models.ParsedItem `json:"observations"`
	}{}

	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, err
	}

	items := append([]models.ParsedItem{}, parsed.Drugs...)
	items = append(items, parsed.LabTests...)
	items = append(items, parsed.Observations...)
	return items, nil
}

func parseAIResponse(req *http.Request) ([]models.ParsedItem, error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("AI status %d: %s", resp.StatusCode, string(b))
	}

	var respdata struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respdata); err != nil {
		return nil, err
	}
	if len(respdata.Choices) == 0 {
		return nil, errors.New("no choices in AI response")
	}

	text := respdata.Choices[0].Message.Content
	parsed := struct {
		Drugs        []models.ParsedItem `json:"drugs"`
		LabTests     []models.ParsedItem `json:"lab_tests"`
		Observations []models.ParsedItem `json:"observations"`
	}{}

	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, err
	}

	items := []models.ParsedItem{}
	items = append(items, parsed.Drugs...)
	items = append(items, parsed.LabTests...)
	items = append(items, parsed.Observations...)
	return items, nil
}

func getBilling(c *gin.Context) {
	patientID, err := strconv.Atoi(c.Param("patient_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient_id"})
		return
	}

	var records []models.MedicalRecord
	if err := DB.Preload("ParsedItems").Where("patient_id = ?", patientID).Find(&records).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load medical records"})
		return
	}

	grandTotal := 0.0
	for _, r := range records {
		grandTotal += r.TotalBill
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"records": records,
		"grand_total": grandTotal,
	})
}

func createPatient(c *gin.Context) {
	var patient models.Patient
	if err := c.ShouldBindJSON(&patient); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := DB.Create(&patient).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create patient"})
		return
	}

	c.JSON(http.StatusCreated, patient)
}

func getPatients(c *gin.Context) {
	var patients []models.Patient
	if err := DB.Find(&patients).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch patients"})
		return
	}

	c.JSON(http.StatusOK, patients)
}

func extractDosage(input string) string {
	re := regexp.MustCompile(`(?i)\b\d+(?:\.\d+)?\s*(mg|ml|tablet|tab|capsule|caps|units)\b`)
	found := re.FindString(input)
	return found
}

func seedDefaultPriceList() {
	defaults := []models.PriceList{
		{ItemName: "Paracetamol", Category: "Drug", Price: 5.00},
		{ItemName: "Amoxicillin", Category: "Drug", Price: 15.00},
		{ItemName: "Full Blood Count", Category: "Lab Test", Price: 25.00},
		{ItemName: "X-Ray Chest", Category: "Lab Test", Price: 50.00},
	}

	for _, p := range defaults {
		if err := DB.FirstOrCreate(&models.PriceList{}, models.PriceList{ItemName: p.ItemName}).Error; err != nil {
			log.Printf("Warning: failed to seed price list item %s: %v", p.ItemName, err)
			continue
		}
		DB.Model(&models.PriceList{}).Where("item_name = ?", p.ItemName).Updates(models.PriceList{Category: p.Category, Price: p.Price})
	}
}

func updatePatient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient id"})
		return
	}

	var patient models.Patient
	if err := c.ShouldBindJSON(&patient); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	patient.ID = uint(id)
	if err := DB.Save(&patient).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update patient"})
		return
	}

	c.JSON(http.StatusOK, patient)
}

func deletePatient(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient id"})
		return
	}

	if err := DB.Delete(&models.Patient{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete patient"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "patient deleted successfully"})
}

func updateMedicalRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	var req ParseNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var record models.MedicalRecord
	if err := DB.First(&record, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "medical record not found"})
		return
	}

	// Update the record
	record.RawNote = req.RawNote
	record.ExtraCharge = req.ExtraCharge

	// Re-parse the note to update parsed items
	priceList := []models.PriceList{}
	if err := DB.Find(&priceList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load price list"})
		return
	}

	// Delete existing parsed items
	if err := DB.Where("record_id = ?", id).Delete(&models.ParsedItem{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete existing parsed items"})
		return
	}

	// Re-parse
	lowerRaw := strings.ToLower(req.RawNote)
	parsed := []models.ParsedItem{}
	seenItems := map[string]bool{}

	for _, p := range priceList {
		if strings.Contains(lowerRaw, strings.ToLower(p.ItemName)) {
			if seenItems[p.ItemName] {
				continue
			}
			dosage := ""
			if strings.EqualFold(p.Category, "Drug") {
				dosage = extractDosage(req.RawNote)
			}
			item := models.ParsedItem{
				RecordID:   record.ID,
				Category:   p.Category,
				ItemName:   p.ItemName,
				Dosage:     dosage,
				Price:      p.Price,
				Confidence: 1.0,
			}
			parsed = append(parsed, item)
			seenItems[p.ItemName] = true
		}
	}

	// Add fallback if no items
	if len(parsed) == 0 {
		lower := strings.ToLower(req.RawNote)
		confidence := 0.3
		if strings.Contains(lower, "pain") || strings.Contains(lower, "cough") || strings.Contains(lower, "fever") {
			confidence = 0.75
		}
		parsed = append(parsed, models.ParsedItem{
			RecordID:   record.ID,
			Category:   "Observation",
			ItemName:   "Clinical Notes",
			Dosage:     "",
			Price:      0,
			Confidence: confidence,
		})
	}

	// Insert new parsed items
	for i := range parsed {
		if err := DB.Create(&parsed[i]).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save parsed item"})
			return
		}
	}

	// Recalculate total
	total := 0.0
	for _, item := range parsed {
		total += item.Price
	}
	total += record.ExtraCharge

	record.TotalBill = total
	if err := DB.Save(&record).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update medical record"})
		return
	}

	c.JSON(http.StatusOK, ParseNoteResponse{
		RecordID:    record.ID,
		PatientID:   record.PatientID,
		TotalBill:   total,
		ExtraCharge: record.ExtraCharge,
		ParsedItems: parsed,
		Notes:       record.RawNote,
	})
}

func deleteMedicalRecord(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid record id"})
		return
	}

	// Delete parsed items first
	if err := DB.Where("record_id = ?", id).Delete(&models.ParsedItem{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete parsed items"})
		return
	}

	// Delete the record
	if err := DB.Delete(&models.MedicalRecord{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete medical record"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "medical record deleted successfully"})
}
