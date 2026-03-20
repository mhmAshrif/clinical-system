import { useState, useEffect } from 'react'
import './App.css'

function App() {
  const [patientId, setPatientId] = useState('')
  const [rawNote, setRawNote] = useState('')
  const [extraCharge, setExtraCharge] = useState('0')
  const [result, setResult] = useState(null)
  const [loading, setLoading] = useState(false)
  const [billing, setBilling] = useState(null)
  const [error, setError] = useState(null)
  const [isListening, setIsListening] = useState(false)
  const [patients, setPatients] = useState([])
  const [selectedPatient, setSelectedPatient] = useState(null)
  const [showPatientForm, setShowPatientForm] = useState(false)
  const [newPatient, setNewPatient] = useState({ name: '', age: '', gender: '' })

  // Load patients on component mount
  useEffect(() => {
    loadPatients()
  }, [])

  const loadPatients = async () => {
    try {
      const response = await fetch('http://localhost:8080/patients')
      if (response.ok) {
        const data = await response.json()
        setPatients(data)
      }
    } catch (error) {
      console.log('Patients endpoint not available yet:', error.message)
    }
  }

  const handleSubmit = async (e) => {
    e.preventDefault()
    if (!patientId || !rawNote.trim()) {
      alert('Please enter both Patient ID and clinic notes')
      return
    }

    setLoading(true)
    setResult(null)
    setError(null)

    try {
      const response = await fetch('http://localhost:8080/parse-note', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          patient_id: parseInt(patientId),
          raw_note: rawNote.trim(),
          extra_charge: parseFloat(extraCharge) || 0
        })
      })

      const responseBody = await response.json().catch(() => null)
      if (!response.ok) {
        const message = responseBody?.error || response.statusText || 'Unknown error'
        throw new Error(message)
      }

      setResult(responseBody)

      // Reload billing after new record
      await handleBilling()

    } catch (err) {
      console.error('Error:', err)
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }

  const handleBilling = async () => {
    if (!patientId) return

    try {
      const response = await fetch(`http://localhost:8080/billing/${patientId}`)
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`)
      }
      const data = await response.json()
      setBilling(data)
    } catch (err) {
      console.error('Error loading billing:', err)
      setError(err.message || 'Failed to load billing history')
    }
  }

  const startVoiceInput = () => {
    if (!('webkitSpeechRecognition' in window) && !('SpeechRecognition' in window)) {
      alert('Voice input not supported in this browser. Please use Chrome or Edge.')
      return
    }

    const SpeechRecognition = window.SpeechRecognition || window.webkitSpeechRecognition
    const recognition = new SpeechRecognition()

    recognition.continuous = false
    recognition.interimResults = false
    recognition.lang = 'en-US'

    recognition.onstart = () => {
      setIsListening(true)
    }

    recognition.onend = () => {
      setIsListening(false)
    }

    recognition.onresult = (event) => {
      const transcript = event.results[0][0].transcript
      setRawNote(prev => prev ? prev + ' ' + transcript : transcript)
    }

    recognition.onerror = (event) => {
      console.error('Speech recognition error:', event.error)
      setIsListening(false)
      alert('Voice recognition failed. Please try again or type manually.')
    }

    try {
      recognition.start()
    } catch (error) {
      console.error('Failed to start recognition:', error)
      alert('Could not start voice recognition')
    }
  }

  const printReport = () => {
    if (!result) return

    const printWindow = window.open('', '_blank')
    const printContent = `
      <!DOCTYPE html>
      <html>
        <head>
          <title>Clinic Report - Patient ${result.patient_id}</title>
          <style>
            body { font-family: Arial, sans-serif; margin: 20px; }
            .header { text-align: center; border-bottom: 2px solid #333; padding-bottom: 10px; margin-bottom: 20px; }
            .section { margin-bottom: 20px; }
            .item { margin: 5px 0; padding: 5px; border-left: 3px solid #007bff; }
            .total { font-size: 18px; font-weight: bold; text-align: right; margin-top: 20px; }
            .category { font-weight: bold; color: #007bff; }
          </style>
        </head>
        <body>
          <div class="header">
            <h1>ABC Health Clinic</h1>
            <h2>Medical Report</h2>
            <p><strong>Patient ID:</strong> ${result.patient_id}</p>
            <p><strong>Date:</strong> ${new Date().toLocaleDateString()}</p>
          </div>

          <div class="section">
            <h3>Clinical Notes:</h3>
            <p>${result.notes}</p>
          </div>

          <div class="section">
            <h3>Prescribed Items:</h3>
            ${result.parsed_items.map(item => `
              <div class="item">
                <span class="category">${item.category}:</span> ${item.item_name}
                ${item.dosage ? ` (${item.dosage})` : ''}
                <span style="float: right;">$${item.price}</span>
              </div>
            `).join('')}
          </div>

          <div class="total">
            Extra Charge: $${result.extra_charge?.toFixed(2) ?? '0.00'}
          </div>
          <div class="total">
            Total Bill: $${result.total_bill.toFixed(2)}
          </div>
        </body>
      </html>
    `

    printWindow.document.write(printContent)
    printWindow.document.close()
    printWindow.print()
  }

  const createPatient = async () => {
    if (!newPatient.name || !newPatient.age || !newPatient.gender) {
      alert('Please fill all patient details')
      return
    }

    try {
      const payload = {
        name: newPatient.name,
        age: parseInt(newPatient.age, 10),
        gender: newPatient.gender,
      }
      console.log('Creating patient with payload:', payload)

      const response = await fetch('http://localhost:8080/patients', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      })

      const data = await response.json().catch(() => null)
      if (!response.ok) {
        console.error('Create patient response failure:', response.status, data)
        const msg = (data && data.error) ? data.error : response.statusText
        alert(`Failed to create patient: ${msg}`)
        return
      }

      console.log('Patient created successfully:', data)
      setPatients([...patients, data])
      setPatientId(data.id.toString())
      setShowPatientForm(false)
      setNewPatient({ name: '', age: '', gender: '' })
      alert('Patient created successfully!')

      // Reload patient list in UI
      loadPatients()
    } catch (error) {
      console.error('Error creating patient:', error)
      alert(`Failed to create patient: ${error.message}`)
    }
  }

  const clearForm = () => {
    setPatientId('')
    setRawNote('')
    setExtraCharge('0')
    setResult(null)
    setBilling(null)
    setError(null)
  }

  return (
    <div className="app">
      <header className="app-header">
        <div className="header-content">
          <h1>🏥 ABC Health Clinic</h1>
          <p>AI-Powered Unified Clinic Notes & Billing System</p>
        </div>
      </header>

      <main className="main-content">
        <div className="container">
          {/* Patient Selection */}
          <section className="section patient-section">
            <h2>👤 Patient Information</h2>
            <div className="patient-controls">
              <div className="form-group">
                <label>Patient ID:</label>
                <select
                  value={patientId}
                  onChange={(e) => {
                    setPatientId(e.target.value)
                    setSelectedPatient(patients.find(p => p.id.toString() === e.target.value))
                  }}
                >
                  <option value="">Select Patient</option>
                  {patients.map(patient => (
                    <option key={patient.id} value={patient.id}>
                      {patient.id} - {patient.name} ({patient.age}y, {patient.gender})
                    </option>
                  ))}
                </select>
              </div>
              <button
                type="button"
                onClick={() => setShowPatientForm(true)}
                className="btn-secondary"
              >
                ➕ New Patient
              </button>
            </div>

            {selectedPatient && (
              <div className="patient-info">
                <h3>Selected Patient: {selectedPatient.name}</h3>
                <p>Age: {selectedPatient.age} | Gender: {selectedPatient.gender}</p>
              </div>
            )}
          </section>

          {/* New Patient Form */}
          {showPatientForm && (
            <div className="modal-overlay">
              <div className="modal">
                <h3>Create New Patient</h3>
                <div className="form-group">
                  <label>Name:</label>
                  <input
                    type="text"
                    value={newPatient.name}
                    onChange={(e) => setNewPatient({...newPatient, name: e.target.value})}
                    placeholder="Patient full name"
                  />
                </div>
                <div className="form-group">
                  <label>Age:</label>
                  <input
                    type="number"
                    value={newPatient.age}
                    onChange={(e) => setNewPatient({...newPatient, age: e.target.value})}
                    placeholder="Age in years"
                  />
                </div>
                <div className="form-group">
                  <label>Gender:</label>
                  <select
                    value={newPatient.gender}
                    onChange={(e) => setNewPatient({...newPatient, gender: e.target.value})}
                  >
                    <option value="">Select Gender</option>
                    <option value="Male">Male</option>
                    <option value="Female">Female</option>
                    <option value="Other">Other</option>
                  </select>
                </div>
                <div className="modal-actions">
                  <button onClick={createPatient} className="btn-primary">Create Patient</button>
                  <button onClick={() => setShowPatientForm(false)} className="btn-secondary">Cancel</button>
                </div>
              </div>
            </div>
          )}

          {error && (
            <div className="alert-error">
              <strong>Error:</strong> {error}
            </div>
          )}

          {/* Clinic Notes Input */}
          <section className="section notes-section">
            <h2>📝 Clinic Notes</h2>
            <form onSubmit={handleSubmit}>
              <div className="form-group">
                <label>Doctor's Notes:</label>
                <textarea
                  value={rawNote}
                  onChange={(e) => setRawNote(e.target.value)}
                  placeholder="Enter all clinic information here (prescriptions, tests, observations)..."
                  rows={8}
                  required
                />
                <div className="form-group">
                  <label>Extra Charge (e.g., travel, consultation fees):</label>
                  <input
                    type="number"
                    step="0.01"
                    min="0"
                    value={extraCharge}
                    onChange={(e) => setExtraCharge(e.target.value)}
                    placeholder="0.00"
                  />
                </div>
                <div className="input-actions">
                  <button
                    type="button"
                    onClick={startVoiceInput}
                    disabled={isListening}
                    className="btn-voice"
                  >
                    {isListening ? '🎤 Listening...' : '🎤 Voice Input'}
                  </button>
                  <button type="button" onClick={clearForm} className="btn-clear">
                    🗑️ Clear
                  </button>
                </div>
              </div>

              <div className="form-actions">
                <button type="submit" disabled={loading || !patientId} className="btn-primary">
                  {loading ? '🔄 Processing...' : '🤖 Parse & Save'}
                </button>
              </div>
            </form>
          </section>

          {/* Results Display */}
          {result && (
            <section className="section results-section">
              <h2>📋 Parsed Results</h2>
              <div className="results-card">
                <div className="results-header">
                  <h3>Record #{result.record_id}</h3>
                  <button onClick={printReport} className="btn-print">
                    🖨️ Print Report
                  </button>
                </div>

                <div className="parsed-items">
                  {result.parsed_items.map((item, index) => (
                    <div key={index} className={`item item-${item.category.toLowerCase().replace(' ', '-')}`}>
                      <div className="item-header">
                        <span className="category">{item.category}</span>
                        <span className="price">${item.price}</span>
                      </div>
                      <div className="item-details">
                        <strong>{item.item_name}</strong>
                        {item.dosage && <span className="dosage">({item.dosage})</span>}
                      </div>
                    </div>
                  ))}
                </div>

                <div className="total-bill">
                  <strong>Extra Charge: ${result.extra_charge?.toFixed(2) ?? '0.00'}</strong>
                </div>
                <div className="total-bill">
                  <strong>Total Bill (incl. extras): ${result.total_bill.toFixed(2)}</strong>
                </div>
              </div>
            </section>
          )}

          {/* Billing History */}
          <section className="section billing-section">
            <h2>💰 Billing History</h2>
            <button
              onClick={handleBilling}
              disabled={!patientId}
              className="btn-secondary"
            >
              📊 Load Billing History
            </button>

            {billing && (
              <div className="billing-card">
                <h3>Patient {billing.patient_id} - Total: ${billing.grand_total}</h3>
                <div className="billing-records">
                  {billing.records.map((record, index) => (
                    <div key={index} className="billing-record">
                      <div className="record-header">
                        <span>Record #{record.id}</span>
                        <span>${record.total_bill?.toFixed(2) ?? '0.00'}</span>
                      </div>
                      <div className="record-meta">
                        <small>Extra Charge: ${record.extra_charge?.toFixed(2) ?? '0.00'}</small>
                        <small>Created: {new Date(record.created_at).toLocaleString()}</small>
                      </div>
                      <p className="record-note">{record.raw_note}</p>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </section>
        </div>
      </main>

      <footer className="app-footer">
        <p>© 2026 ABC Health Clinic - AI-Powered Medical System</p>
      </footer>
    </div>
  )
}

export default App
