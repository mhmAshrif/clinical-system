-- 1. Create Patients Table
CREATE TABLE IF NOT EXISTS patients (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    age INT,
    gender TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 2. Create Medical Records (Main table for the doctor's note)
CREATE TABLE IF NOT EXISTS medical_records (
    id SERIAL PRIMARY KEY,
    patient_id INT REFERENCES patients(id),
    raw_note TEXT NOT NULL,
    total_bill DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Create Parsed Items (Categorized by AI)
CREATE TABLE IF NOT EXISTS parsed_items (
    id SERIAL PRIMARY KEY,
    record_id INT REFERENCES medical_records(id) ON DELETE CASCADE,
    category TEXT,
    item_name TEXT NOT NULL,
    dosage TEXT,
    price DECIMAL(10,2) DEFAULT 0
);

-- 4. Create a basic Price List (For Billing Logic)
CREATE TABLE IF NOT EXISTS price_list (
    item_name TEXT PRIMARY KEY,
    category TEXT,
    price DECIMAL(10,2)
);

-- Insert sample prices for testing
INSERT INTO price_list (item_name, category, price) VALUES
('Paracetamol', 'Drug', 5.00),
('Amoxicillin', 'Drug', 15.00),
('Full Blood Count', 'Lab Test', 25.00),
('X-Ray Chest', 'Lab Test', 50.00)
ON CONFLICT (item_name) DO UPDATE SET category = EXCLUDED.category, price = EXCLUDED.price;
