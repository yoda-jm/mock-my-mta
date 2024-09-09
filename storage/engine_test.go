package storage

import (
	"net/mail"
	"strings"
	"testing"
	"time"
)

func TestUnimplementedMethodInLayerError(t *testing.T) {
	// check that error message contains the method name and the layer name
	err := newUnimplementedMethodInLayerError("my-method", "my-layer")
	if !strings.Contains(err.Error(), "my-method") {
		t.Errorf("Expected error message to contain method name, got %v", err.Error())
	}
	if !strings.Contains(err.Error(), "my-layer") {
		t.Errorf("Expected error message to contain layer name, got %v", err.Error())
	}
}

func TestNewEngineUnknownStorageLayerType(t *testing.T) {
	// Test that NewEngine returns a valid Engine
	storageConfiguration := []StorageLayerConfiguration{
		{
			Type: "unknown-storage-layer-type",
			Parameters: map[string]string{
				"key": "value",
			},
		},
	}

	_, err := NewEngine(storageConfiguration)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestNewEngineOneStorageLayer(t *testing.T) {
	// Test that NewEngine returns a valid Engine
	storageConfigurations := [][]StorageLayerConfiguration{
		{
			{
				Type:       "MEMORY",
				Parameters: map[string]string{},
			},
		},
		{
			{
				Type: "SQLITE",
				Parameters: map[string]string{
					"database": "test.db",
				},
			},
		},
		{
			{
				Type: "FILESYSTEM",
				Parameters: map[string]string{
					"folder": "test-folder",
					"type":   "eml",
				},
			},
		},
	}

	for _, storageConfiguration := range storageConfigurations {
		engine, err := NewEngine(storageConfiguration)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		// check that there is exactly one storage layer
		if len(engine.storages) != 1 {
			t.Errorf("Expected exactly one storage layer, got %v", len(engine.storages))
		}
		// check that the storage layer is of the correct type
		storageLayerConfiguration := storageConfiguration[0]
		switch storageLayerConfiguration.Type {
		case "MEMORY":
			// check that the storage layer is of the correct type
			if _, ok := engine.storages[0].(*memoryStorage); !ok {
				t.Errorf("Expected MemoryStorage, got %T", engine.storages[0])
			}
		case "SQLITE":
			// check that the storage layer is of the correct type
			if storage, ok := engine.storages[0].(*sqliteStorage); !ok {
				t.Errorf("Expected SqliteStorage, got %T", engine.storages[0])
			} else {
				// check that the database file is correct
				if storage.databaseFilename != storageLayerConfiguration.Parameters["database"] {
					t.Errorf("Expected test.db, got %v", storage.databaseFilename)
				}
			}
		case "FILESYSTEM":
			// check that the storage layer is of the correct type
			if storage, ok := engine.storages[0].(*filesystemStorage); !ok {
				t.Errorf("Expected FilesystemStorage, got %T", engine.storages[0])
			} else {
				// check that the folder is correct
				if storage.folder != storageLayerConfiguration.Parameters["folder"] {
					t.Errorf("Expected test-folder, got %v", storage.folder)
				}
			}
		default:
			t.Errorf("Unknown storage layer type: %v", storageConfiguration[0].Type)
		}
	}
}

func TestEngineSetWithNoDate(t *testing.T) {
	// Test that Engine.Set calls the correct method in the storage layer
	engine := &Engine{
		storages: []storageLayer{newMockStorageLayer(getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods))},
	}
	err := engine.Set(&mail.Message{Header: mail.Header{}})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// check that the correct method has been called
	mockStorageLayer := engine.storages[0].(*mockStorageLayer)
	if _, found := mockStorageLayer.calls["setWithID"]; !found {
		t.Errorf("Expected setWithID to be called")
	}
	// check that the correct arguments have been passed
	if len(mockStorageLayer.calls["setWithID"]) != 2 {
		t.Errorf("Expected two arguments, got %v", len(mockStorageLayer.calls["setWithID"]))
	}
	if mockStorageLayer.calls["setWithID"][0] == "" {
		t.Errorf("Expected to pass email ID, got empty string")
	}
	if mockStorageLayer.calls["setWithID"][1] == nil {
		t.Errorf("Expected non-nil message, got nil")
	}
	email := mockStorageLayer.calls["setWithID"][1].(*mail.Message)
	if email.Header == nil {
		t.Errorf("Expected non-nil header, got nil")
	}
	// expect that the email header to have a date
	if email.Header.Get("Date") == "" {
		t.Errorf("Expected email header to have a date, got empty string")
	}
	// check that the date is a valid RFC1123Z date
	_, err = time.Parse(time.RFC1123Z, email.Header.Get("Date"))
	if err != nil {
		t.Errorf("Expected email header to have a valid date, got %v", email.Header.Get("Date"))
	}
}

func TestEngineSetWithDate(t *testing.T) {
	// Test that Engine.Set calls the correct method in the storage layer
	engine := &Engine{
		storages: []storageLayer{newMockStorageLayer(getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods))},
	}
	date := time.Now()
	err := engine.Set(&mail.Message{Header: mail.Header{"Date": []string{date.Format(time.RFC1123Z)}}})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// check that the correct method has been called
	mockStorageLayer := engine.storages[0].(*mockStorageLayer)
	if _, found := mockStorageLayer.calls["setWithID"]; !found {
		t.Errorf("Expected setWithID to be called")
	}
	// check that the correct arguments have been passed
	if len(mockStorageLayer.calls["setWithID"]) != 2 {
		t.Errorf("Expected two arguments, got %v", len(mockStorageLayer.calls["setWithID"]))
	}
	if mockStorageLayer.calls["setWithID"][0] == "" {
		t.Errorf("Expected to pass email ID, got empty string")
	}
	if mockStorageLayer.calls["setWithID"][1] == nil {
		t.Errorf("Expected non-nil message, got nil")
	}
	email := mockStorageLayer.calls["setWithID"][1].(*mail.Message)
	if email.Header == nil {
		t.Errorf("Expected non-nil header, got nil")
	}
	// expect that the email header have the correct date
	if email.Header.Get("Date") != date.Format(time.RFC1123Z) {
		t.Errorf("Expected email header to have date %v, got %v", date.Format(time.RFC1123Z), email.Header.Get("Date"))
	}
}

func TestEngineSetWithInvalidDate(t *testing.T) {
	// Test that Engine.Set set current date when the date is invalid
	engine := &Engine{
		storages: []storageLayer{newMockStorageLayer(getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods))},
	}
	err := engine.Set(&mail.Message{Header: mail.Header{"Date": []string{"invalid-date"}}})
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// check that the correct method has been called
	mockStorageLayer := engine.storages[0].(*mockStorageLayer)
	if _, found := mockStorageLayer.calls["setWithID"]; !found {
		t.Errorf("Expected setWithID to be called")
	}
	// check that the correct arguments have been passed
	if len(mockStorageLayer.calls["setWithID"]) != 2 {
		t.Errorf("Expected two arguments, got %v", len(mockStorageLayer.calls["setWithID"]))
	}
	if mockStorageLayer.calls["setWithID"][0] == "" {
		t.Errorf("Expected to pass email ID, got empty string")
	}
	if mockStorageLayer.calls["setWithID"][1] == nil {
		t.Errorf("Expected non-nil message, got nil")
	}
	email := mockStorageLayer.calls["setWithID"][1].(*mail.Message)
	if email.Header == nil {
		t.Errorf("Expected non-nil header, got nil")
	}
	// expect that the email header to have a date
	if email.Header.Get("Date") == "" {
		t.Errorf("Expected email header to have a date, got empty string")
	}
	// check that the date is a valid RFC1123Z date
	_, err = time.Parse(time.RFC1123Z, email.Header.Get("Date"))
	if err != nil {
		t.Errorf("Expected email header to have a valid date, got %v", email.Header.Get("Date"))
	}
}

func TestNewEngineThreeStorageLayers(t *testing.T) {
	// Test that NewEngine returns a valid Engine
	storageConfiguration := []StorageLayerConfiguration{
		{
			Type:       "MEMORY",
			Parameters: map[string]string{},
		},
		{
			Type: "SQLITE",
			Parameters: map[string]string{
				"database": "test.db",
			},
		},
		{
			Type: "FILESYSTEM",
			Parameters: map[string]string{
				"folder": "test-folder",
				"type":   "eml",
			},
		},
	}

	engine, err := NewEngine(storageConfiguration)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	// check that there are three storage layers
	if len(engine.storages) != 3 {
		t.Errorf("Expected three storage layers, got %v", len(engine.storages))
	}
	// check that the storage layers are of the correct type
	if _, ok := engine.storages[0].(*memoryStorage); !ok {
		t.Errorf("Expected MemoryStorage, got %T", engine.storages[0])
	}
	if _, ok := engine.storages[1].(*sqliteStorage); !ok {
		t.Errorf("Expected SqliteStorage, got %T", engine.storages[1])
	}
	if _, ok := engine.storages[2].(*filesystemStorage); !ok {
		t.Errorf("Expected FilesystemStorage, got %T", engine.storages[2])
	}
}

func TestNewEngineMissingParameter(t *testing.T) {
	// Test that NewEngine returns an error when a parameter is missing
	storageConfigurations := [][]StorageLayerConfiguration{
		{
			{
				Type: "SQLITE",
				Parameters: map[string]string{
					"missing-parameter": "test.db",
				},
			},
		},
		{
			{
				Type: "FILESYSTEM",
				Parameters: map[string]string{
					"missing-parameter": "test-folder",
				},
			},
		},
	}

	for _, storageConfiguration := range storageConfigurations {
		_, err := NewEngine(storageConfiguration)
		if err == nil {
			t.Errorf("Expected error, got nil")
		}
	}
}

type mockConfigurationType int

const (
	mockConfigurationTypeNoUnimplementedMethods mockConfigurationType = iota
	mockConfigurationTypeUnimplementedGetMailboxesMethod
	mockConfigurationTypeAllUnimplementedMethods
)

func getMockConfiguration(t mockConfigurationType) map[string]bool {
	switch t {
	case mockConfigurationTypeNoUnimplementedMethods:
		return map[string]bool{}
	case mockConfigurationTypeUnimplementedGetMailboxesMethod:
		return map[string]bool{
			"GetMailboxes": true,
		}
	case mockConfigurationTypeAllUnimplementedMethods:
		return map[string]bool{
			// "load" is mandatory
			"setWithID":       true,
			"DeleteEmailByID": true,
			"GetAttachment":   true,
			"GetAttachments":  true,
			"GetBodyVersion":  true,
			"GetEmailByID":    true,
			"GetMailboxes":    true,
			"SearchEmails":    true,
		}
	}
	panic("Unknown mock configuration type")
}

func getAllMethods() []string {
	return []string{
		"load",
		"setWithID",
		"DeleteEmailByID",
		"GetAttachment",
		"GetAttachments",
		"GetBodyVersion",
		"GetEmailByID",
		"GetMailboxes",
		"SearchEmails",
	}
}

func executeAllEngineMethods(engine *Engine) map[string]error {
	errors := make(map[string]error)
	errors["load"] = engine.load(nil)
	errors["setWithID"] = engine.setWithID("email-id", &mail.Message{})
	errors["DeleteEmailByID"] = engine.DeleteEmailByID("email-id")
	_, errors["GetAttachment"] = engine.GetAttachment("email-id", "attachment-id")
	_, errors["GetAttachments"] = engine.GetAttachments("email-id")
	_, errors["GetBodyVersion"] = engine.GetBodyVersion("email-id", EmailVersionRaw)
	_, errors["GetEmailByID"] = engine.GetEmailByID("email-id")
	_, errors["GetMailboxes"] = engine.GetMailboxes()
	_, _, errors["SearchEmails"] = engine.SearchEmails("query", 1, 10)
	return errors
}

func TestMockStorageLayerEngineOneLayer(t *testing.T) {
	// Test that the mock storage layer is correctly used in the engine
	mockConfigurations := []struct {
		// map of method names that are unimplemented
		unimplementedCalls map[string]bool
	}{
		{
			// no unimplemented methods
			unimplementedCalls: getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods),
		},
		{
			// unimplemented method
			unimplementedCalls: getMockConfiguration(mockConfigurationTypeUnimplementedGetMailboxesMethod),
		},
		{
			// all methods are unimplemented
			unimplementedCalls: getMockConfiguration(mockConfigurationTypeAllUnimplementedMethods),
		},
	}
	for _, mockConfiguration := range mockConfigurations {
		mockStorageLayer := newMockStorageLayer(mockConfiguration.unimplementedCalls)
		engine := &Engine{
			storages: []storageLayer{mockStorageLayer},
		}
		errors := executeAllEngineMethods(engine)
		// check that all methods have been called
		for _, methodName := range getAllMethods() {
			if _, found := mockStorageLayer.calls[methodName]; !found {
				t.Errorf("Expected %v to be called", methodName)
			}
		}
		// check that the errors are correctly returned
		for methodName, err := range errors {
			// check that method is called
			if _, found := mockStorageLayer.calls[methodName]; !found {
				t.Errorf("Expected %v to be called", methodName)
			}
			switch methodName {
			case "DeleteEmailByID", "setWithID":
				// DeleteEmailByID and setWithID should not return an error in case of unimplemented method
				if err != nil {
					t.Errorf("Unexpected error for method %v: %v", methodName, err)
				}
			default:
				// check that the error is correctly returned
				if mockConfiguration.unimplementedCalls[methodName] && err == nil {
					t.Errorf("Expected error for unimplemented method %v, got nil", methodName)
				}
				if !mockConfiguration.unimplementedCalls[methodName] && err != nil {
					t.Errorf("Unexpected error for method %v: %v", methodName, err)
				}
			}
		}
	}
}

func TestMockStorageLayerEngineTwoLayersFullDefaulting(t *testing.T) {
	// Test that the mock storage layer is correctly used in the engine
	layer1 := newMockStorageLayer(getMockConfiguration(mockConfigurationTypeAllUnimplementedMethods))
	layer2 := newMockStorageLayer(getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods))
	engine := &Engine{
		storages: []storageLayer{layer1, layer2},
	}

	errors := executeAllEngineMethods(engine)
	for _, methodName := range getAllMethods() {
		// check that all methods have been called
		switch methodName {
		case "load", "setWithID", "DeleteEmailByID", "GetAttachment", "GetAttachments", "GetBodyVersion", "GetEmailByID", "SearchEmails", "GetMailboxes":
			// should be called in the first layer
			if _, found := layer1.calls[methodName]; !found {
				t.Errorf("Expected %v to be called", methodName)
			}
			// should be called in the second layer (defaulting)
			if _, found := layer2.calls[methodName]; !found {
				t.Errorf("Expected %v to be called", methodName)
			}
		default:
			t.Errorf("Expected %v to be called", methodName)
		}
	}
	// check that the errors are correctly returned
	for methodName, err := range errors {
		// no error should be returned because the method is unimplemented in at least one layer
		if err != nil {
			t.Errorf("Unexpected error for method %v: %v", methodName, err)
		}
	}
}

func TestMockStorageLayerEngineTwoLayersNoDefaulting(t *testing.T) {
	// Test that the mock storage layer is correctly used in the engine
	layer1 := newMockStorageLayer(getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods))
	layer2 := newMockStorageLayer(getMockConfiguration(mockConfigurationTypeNoUnimplementedMethods))
	engine := &Engine{
		storages: []storageLayer{layer1, layer2},
	}

	errors := executeAllEngineMethods(engine)
	for _, methodName := range getAllMethods() {
		// check that all methods have been called
		switch methodName {
		case "load", "setWithID", "DeleteEmailByID":
			// those methods should be called in all layers
			if _, found := layer1.calls[methodName]; !found {
				t.Errorf("Expected %v to be called in layer 1", methodName)
			}
			if _, found := layer2.calls[methodName]; !found {
				t.Errorf("Expected %v to be called in layer 2", methodName)
			}
		case "GetAttachment", "GetAttachments", "GetBodyVersion", "GetEmailByID", "SearchEmails", "GetMailboxes":
			// should be called in the first layer
			if _, found := layer1.calls[methodName]; !found {
				t.Errorf("Expected %v to be called in layer 1", methodName)
			}
			// should not be called in the second layer
			if _, found := layer2.calls[methodName]; found {
				t.Errorf("Expected %v to be called in layer 2", methodName)
			}
		default:
			t.Errorf("Expected %v to be called", methodName)
		}
	}

	// check that the errors are correctly returned
	for methodName, err := range errors {
		// there should be no error (all methods are implemented in at least one layer)
		if err != nil {
			t.Errorf("Unexpected error for method %v: %v", methodName, err)
		}
	}
}

type mockStorageLayer struct {
	// list of method names that have been called with their arguments
	calls              map[string][]interface{}
	unimplementedCalls map[string]bool
}

// check that mockStorageLayer implements the StorageLayer interface
var _ storageLayer = &mockStorageLayer{}

func newMockStorageLayer(unimplementedCalls map[string]bool) *mockStorageLayer {
	return &mockStorageLayer{unimplementedCalls: unimplementedCalls}
}

func (m *mockStorageLayer) addCall(methodName string, args ...interface{}) error {
	if m.calls == nil {
		m.calls = make(map[string][]interface{})
	}
	m.calls[methodName] = append(m.calls[methodName], args...)
	if _, found := m.unimplementedCalls[methodName]; found {
		return newUnimplementedMethodInLayerError(methodName, "mockStorageLayer")
	}
	return nil
}

func (m *mockStorageLayer) DeleteEmailByID(emailID string) error {
	return m.addCall("DeleteEmailByID", emailID)
}

func (m *mockStorageLayer) DeleteAllEmails() error {
	return m.addCall("DeleteAllEmails")
}

func (m *mockStorageLayer) GetAttachment(emailID string, attachmentID string) (Attachment, error) {
	return Attachment{}, m.addCall("GetAttachment", emailID, attachmentID)
}

func (m *mockStorageLayer) GetAttachments(emailID string) ([]AttachmentHeader, error) {
	return nil, m.addCall("GetAttachments", emailID)
}

func (m *mockStorageLayer) GetBodyVersion(emailID string, version EmailVersionType) (string, error) {
	return "", m.addCall("GetBodyVersion", emailID, version)
}

func (m *mockStorageLayer) GetEmailByID(emailID string) (EmailHeader, error) {
	return EmailHeader{}, m.addCall("GetEmailByID", emailID)
}

func (m *mockStorageLayer) GetMailboxes() ([]Mailbox, error) {
	return nil, m.addCall("GetMailboxes")
}

func (m *mockStorageLayer) SearchEmails(query string, page int, pageSize int) ([]EmailHeader, int, error) {
	return nil, 0, m.addCall("SearchEmails", query, page, pageSize)
}

func (m *mockStorageLayer) load(rootStorage Storage) error {
	return m.addCall("load", rootStorage)
}

func (m *mockStorageLayer) setWithID(emailID string, message *mail.Message) error {
	return m.addCall("setWithID", emailID, message)
}
