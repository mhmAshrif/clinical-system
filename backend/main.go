package main

import (
	"log"
	"net/http"
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
	r.GET("/billing/:patient_id", getBilling)
	r.POST("/patients", createPatient)
	r.GET("/patients", getPatients)

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
				RecordID: record.ID,
				Category: p.Category,
				ItemName: p.ItemName,
				Dosage:   dosage,
				Price:    p.Price,
			}
			parsed = append(parsed, item)
			seenItems[p.ItemName] = true
		}
	}

	// Add a fallback observation entry for remaining text if no parsed items
	if len(parsed) == 0 {
		parsed = append(parsed, models.ParsedItem{
			RecordID: record.ID,
			Category: "Observation",
			ItemName: "Clinical Notes",
			Dosage:   "",
			Price:    0,
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
