package data

import (
	"fmt"
	"log"
	"os"
	"path"
	"sync"

	"github.com/Malpizarr/dbproto/dbdata"
	"github.com/Malpizarr/dbproto/utils"
	"google.golang.org/protobuf/proto"
)

type Record map[string]interface{}

type TableReader interface {
	Select(key string) (Record, error)
	Insert(record Record) error
	Update(key string, record Record) error
	Delete(key string) error
}

type Table struct {
	sync.RWMutex
	FilePath   string
	PrimaryKey string
	utils      *utils.Utils
}

func NewTable(primaryKey, filePath string) *Table {
	dir := path.Dir(filePath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}
	log.Printf("Creating table with file path: %s", filePath)

	table := &Table{
		FilePath:   filePath,
		PrimaryKey: primaryKey,
		utils:      utils.NewUtils(),
	}

	if err := table.initializeFileIfNotExists(); err != nil {
		log.Fatalf("Failed to initialize file %s: %v", filePath, err)
	} else {
		log.Printf("File %s initialized successfully.", filePath)
	}

	return table
}

func (t *Table) initializeFileIfNotExists() error {
	if _, err := os.Stat(t.FilePath); os.IsNotExist(err) {
		records := &dbdata.Records{
			Records: make(map[string]*dbdata.Record),
		}
		if err := t.writeRecordsToFile(records); err != nil {
			return fmt.Errorf("failed to write initial data to file: %v", err)
		}
	}
	return nil
}

func (t *Table) Insert(record Record) error {
	t.Lock()
	defer t.Unlock()

	protoRecords, err := t.readRecordsFromFile()
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%v", record[t.PrimaryKey])
	if _, exists := protoRecords.GetRecords()[key]; exists {
		return fmt.Errorf("Record with key %s already exists", key)
	}

	newProtoRecord := &dbdata.Record{Fields: make(map[string]string)}
	for k, v := range record {
		strVal, ok := v.(string)
		if !ok {
			return fmt.Errorf("non-string value found for key %s: value %v", k, v)
		}
		newProtoRecord.Fields[k] = strVal
	}

	protoRecords.GetRecords()[key] = newProtoRecord
	return t.writeRecordsToFile(protoRecords)
}

func (t *Table) SelectAll() ([]*dbdata.Record, error) {
	t.RLock()
	defer t.RUnlock()
	records, err := t.readRecordsFromFile()
	if err != nil {
		return nil, err
	}
	var allRecords []*dbdata.Record
	for _, record := range records.GetRecords() {
		allRecords = append(allRecords, record)
	}
	return allRecords, nil
}

func (t *Table) Update(key string, updates Record) error {
	t.Lock()
	defer t.Unlock()
	protoRecords, err := t.readRecordsFromFile()
	if err != nil {
		return err
	}
	record, exists := protoRecords.GetRecords()[key]
	if !exists {
		return fmt.Errorf("Record with key %s not found", key)
	}

	for field, value := range updates {
		strVal, ok := value.(string)
		if !ok {
			return fmt.Errorf("non-string value for field %s", field)
		}
		record.Fields[field] = strVal
	}
	return t.writeRecordsToFile(protoRecords)
}

func (t *Table) Delete(key string) error {
	t.Lock()
	defer t.Unlock()
	records, err := t.readRecordsFromFile()
	if err != nil {
		return err
	}
	if _, exists := records.GetRecords()[key]; !exists {
		return fmt.Errorf("Record with key %s not found", key)
	}
	delete(records.GetRecords(), key)
	return t.writeRecordsToFile(records)
}

func (t *Table) readRecordsFromFile() (*dbdata.Records, error) {
	encryptedData, err := os.ReadFile(t.FilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &dbdata.Records{Records: make(map[string]*dbdata.Record)}, nil
		}
		return nil, fmt.Errorf("failed to read file: %v", err)
	}

	if len(encryptedData) == 0 {
		return &dbdata.Records{Records: make(map[string]*dbdata.Record)}, nil
	}

	decryptedData, err := t.utils.Decrypt(string(encryptedData))
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %v", err)
	}

	var records dbdata.Records
	if err := proto.Unmarshal(decryptedData, &records); err != nil {
		return nil, fmt.Errorf("proto unmarshal failed: %v", err)
	}

	return &records, nil
}

func (t *Table) writeRecordsToFile(records *dbdata.Records) error {
	data, err := proto.Marshal(records)
	if err != nil {
		return fmt.Errorf("error marshaling records: %v", err)
	}
	encryptedData, err := t.utils.Encrypt(data)
	if err != nil {
		return fmt.Errorf("error encrypting data: %v", err)
	}

	file, err := os.OpenFile(t.FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening file '%s': %v", t.FilePath, err)
	}
	defer file.Close()

	n, err := file.Write([]byte(encryptedData))
	if err != nil {
		return fmt.Errorf("error writing to file '%s': %v", t.FilePath, err)
	}
	log.Printf("Wrote %d bytes to file %s", n, t.FilePath)

	return nil
}
