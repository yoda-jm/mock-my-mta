package smtp

import (
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"reflect"
	"testing"

	"github.com/chrj/smtpd"
	"mock-my-mta/storage"
)

func TestRelayConfigurations_Names(t *testing.T) {
	tests := []struct {
		name string
		rc   RelayConfigurations
		want []string
	}{
		{name: "no relays", rc: RelayConfigurations{}, want: []string{}},
		{name: "only enabled relays", rc: RelayConfigurations{"relay1": {Enabled: true}, "relay2": {Enabled: true}}, want: []string{"relay1", "relay2"}},
		{name: "ignore disabled relays", rc: RelayConfigurations{"relay1": {Enabled: true}, "relay2": {Enabled: false}, "relay3": {Enabled: true}}, want: []string{"relay1", "relay3"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.rc.Names()
			if !reflect.DeepEqual(sorted(got), sorted(tt.want)) {
				t.Errorf("RelayConfigurations.Names() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLoginAuth(t *testing.T) {
	username, password := "testuser", "testpass"
	auth := newLoginAuth(username, password)
	if auth == nil {
		t.Fatal("newLoginAuth returned nil")
	}
	la, ok := auth.(*loginAuth)
	if !ok {
		t.Fatalf("newLoginAuth did not return a *loginAuth type, got %T", auth)
	}
	if la.username != username || la.password != password {
		t.Errorf("expected username %v, password %v, got %v, %v", username, password, la.username, la.password)
	}
}

func TestLoginAuth_Start(t *testing.T) {
	auth := newLoginAuth("user", "pass").(*loginAuth)
	proto, toServer, err := auth.Start(nil) // serverInfo is not used by this implementation
	if err != nil || proto != "LOGIN" || len(toServer) != 0 {
		t.Errorf("Start() got proto %q, toServer %v, err %v; want proto \"LOGIN\", empty toServer, nil err", proto, toServer, err)
	}
}

func TestLoginAuth_Next(t *testing.T) {
	username, password := "user123", "pass456"
	auth := newLoginAuth(username, password).(*loginAuth)
	tests := []struct {
		name         string
		fromServer   []byte
		more         bool
		wantToServer []byte
		wantErr      bool
	}{
		{"more is false", nil, false, nil, false},
		{"challenge Username", []byte("Username:"), true, []byte(username), false},
		{"challenge Password", []byte("Password:"), true, []byte(password), false},
		{"unexpected challenge", []byte("Unexpected:"), true, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toServer, err := auth.Next(tt.fromServer, tt.more)
			if (err != nil) != tt.wantErr || !reflect.DeepEqual(toServer, tt.wantToServer) {
				t.Errorf("Next(%q, %v) got toServer %v, err %v; want toServer %v, wantErr %v", tt.fromServer, tt.more, toServer, err, tt.wantToServer, tt.wantErr)
			}
		})
	}
}

func TestRelayConfigurations_Get(t *testing.T) {
	rc := RelayConfigurations{
		"enabledRelay":  {Enabled: true, Addr: "addr1"},
		"disabledRelay": {Enabled: false, Addr: "addr2"},
	}
	tests := []struct {
		name      string
		relayName string
		wantRelay RelayConfiguration
		wantOk    bool
	}{
		{"get enabled", "enabledRelay", RelayConfiguration{Enabled: true, Addr: "addr1"}, true},
		{"get disabled", "disabledRelay", RelayConfiguration{}, false},
		{"get non-existent", "noSuchRelay", RelayConfiguration{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRelay, gotOk := rc.Get(tt.relayName)
			if !reflect.DeepEqual(gotRelay, tt.wantRelay) || gotOk != tt.wantOk {
				t.Errorf("Get(%q) got relay %v, ok %v; want relay %v, ok %v", tt.relayName, gotRelay, gotOk, tt.wantRelay, tt.wantOk)
			}
		})
	}
}

// sorted returns a sorted copy of the slice.
func sorted(s []string) []string {
	if s == nil {
		return nil
	}
	c := make([]string, len(s))
	copy(c, s)
	for i := 0; i < len(c); i++ {
		for j := i + 1; j < len(c); j++ {
			if c[i] > c[j] {
				c[i], c[j] = c[j], c[i]
			}
		}
	}
	return c
}

// mockIoStorage is a mock implementation of the storage.StorageService interface.
type mockIoStorage struct {
	SetFn             func(message *mail.Message) (string, error)
	SetError          error
	SetUUID           string
	SetCalled         bool
	LastMessage       *mail.Message
	GetMailboxesFn    func() ([]storage.Mailbox, error)
	GetEmailByIDFn    func(emailID string) (storage.EmailHeader, error)
	DeleteAllEmailsFn func() error
	DeleteEmailByIDFn func(emailID string) error
	GetBodyVersionFn  func(emailID string, version storage.EmailVersionType) (string, error)
	GetAttachmentsFn  func(emailID string) ([]storage.AttachmentHeader, error)
	GetAttachmentFn   func(emailID string, attachmentID string) (storage.Attachment, error)
	SearchEmailsFn    func(query string, page, pageSize int) ([]storage.EmailHeader, int, error)
}

func (m *mockIoStorage) Set(message *mail.Message) (string, error) {
	m.SetCalled = true
	m.LastMessage = message
	if m.SetFn != nil {
		return m.SetFn(message)
	}
	return m.SetUUID, m.SetError
}
func (m *mockIoStorage) GetMailboxes() ([]storage.Mailbox, error) {
	if m.GetMailboxesFn != nil { return m.GetMailboxesFn() }; return nil, nil
}
func (m *mockIoStorage) GetEmailByID(emailID string) (storage.EmailHeader, error) {
	if m.GetEmailByIDFn != nil { return m.GetEmailByIDFn(emailID) }; return storage.EmailHeader{}, nil
}
func (m *mockIoStorage) DeleteAllEmails() error {
	if m.DeleteAllEmailsFn != nil { return m.DeleteAllEmailsFn() }; return nil
}
func (m *mockIoStorage) DeleteEmailByID(emailID string) error {
	if m.DeleteEmailByIDFn != nil { return m.DeleteEmailByIDFn(emailID) }; return nil
}
func (m *mockIoStorage) GetBodyVersion(emailID string, version storage.EmailVersionType) (string, error) {
	if m.GetBodyVersionFn != nil { return m.GetBodyVersionFn(emailID, version) }; return "", nil
}
func (m *mockIoStorage) GetAttachments(emailID string) ([]storage.AttachmentHeader, error) {
	if m.GetAttachmentsFn != nil { return m.GetAttachmentsFn(emailID) }; return nil, nil
}
func (m *mockIoStorage) GetAttachment(emailID string, attachmentID string) (storage.Attachment, error) {
	if m.GetAttachmentFn != nil { return m.GetAttachmentFn(emailID, attachmentID) }; return storage.Attachment{}, nil
}
func (m *mockIoStorage) SearchEmails(query string, page, pageSize int) ([]storage.EmailHeader, int, error) {
	if m.SearchEmailsFn != nil { return m.SearchEmailsFn(query, page, pageSize) }; return nil, 0, nil
}

// Ensure mockIoStorage implements storage.StorageService
var _ storage.StorageService = &mockIoStorage{}

func TestRelayMessage(t *testing.T) {
	originalSendMailFn := smtpSendMailFn
	t.Cleanup(func() { smtpSendMailFn = originalSendMailFn })

	tests := []struct {
		name             string
		relayConfig      RelayConfiguration
		wantErr          bool
		expectedAuthType interface{} 
	}{
		{"success RelayAuthModeNone", RelayConfiguration{Mechanism: RelayAuthModeNone, Addr: "host:25"}, false, nil},
		{"success RelayAuthModePlain", RelayConfiguration{Mechanism: RelayAuthModePlain, Addr: "host:587", Username: "u", Password: "p"}, false, reflect.TypeOf((*smtp.Auth)(nil)).Elem()},
		{"success RelayAuthModeLogin", RelayConfiguration{Mechanism: RelayAuthModeLogin, Addr: "host:587", Username: "u", Password: "p"}, false, reflect.TypeOf(&loginAuth{})},
		{"success RelayAuthModeCramMD5", RelayConfiguration{Mechanism: RelayAuthModeCramMD5, Addr: "host:587", Username: "u", Password: "p"}, false, reflect.TypeOf((*smtp.Auth)(nil)).Elem()},
		{"unsupported auth", RelayConfiguration{Mechanism: "UNKNOWN", Addr: "host:25"}, true, nil},
		{"plain auth missing host port", RelayConfiguration{Mechanism: RelayAuthModePlain, Addr: "invalid"}, true, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smtpSendMailFn = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
				if tt.expectedAuthType != nil {
					actualType := reflect.TypeOf(a)
					if actualType != tt.expectedAuthType {
						// Check if expectedAuthType is an interface (like smtp.Auth)
						if expectedInterfaceType, ok := tt.expectedAuthType.(reflect.Type); ok && expectedInterfaceType.Kind() == reflect.Interface {
							if a == nil || !actualType.Implements(expectedInterfaceType) {
								t.Errorf("auth type %T does not implement expected interface %v", a, expectedInterfaceType)
							}
						} else { // Direct type comparison
							t.Errorf("auth type = %T, want %T", a, tt.expectedAuthType)
						}
					}
				} else if a != nil {
					t.Errorf("auth type = %T, want nil", a)
				}
				return nil
			}
			err := RelayMessage(tt.relayConfig, "uuid", Envelope{Sender: "s", Recipients: []string{"r"}, Data: []byte("d")})
			if (err != nil) != tt.wantErr {
				t.Errorf("RelayMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestServer_handler(t *testing.T) {
	originalSendMailFn := smtpSendMailFn
	t.Cleanup(func() { smtpSendMailFn = originalSendMailFn })

	minimalEmailData := []byte("From: sender@example.com\nTo: recipient@example.com\nSubject: Test Email\n\nThis is a test email.")
	mockPeer := smtpd.Peer{Addr: &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}}

	tests := []struct {
		name                  string
		serverConfig          Configuration
		envelope              smtpd.Envelope
		mockIoStoreSetup      func(*mockIoStorage)
		smtpSendMailFnSetup   func() (sendMailMock func(addr string, a smtp.Auth, from string, to []string, msg []byte) error, calls *int)
		wantErr               bool
		expectedSetCalled     bool
		expectedSendMailCalls int
	}{
		{
			name:         "Successful Email Processing and Storage",
			serverConfig: Configuration{Relays: RelayConfigurations{}},
			envelope:     smtpd.Envelope{Sender: "s@s.com", Recipients: []string{"r@r.com"}, Data: minimalEmailData},
			mockIoStoreSetup: func(ms *mockIoStorage) {
				ms.SetUUID = "test-uuid-1"; ms.SetError = nil
			},
			smtpSendMailFnSetup: func() (func(string, smtp.Auth, string, []string, []byte) error, *int) {
				calls := 0; return func(string, smtp.Auth, string, []string, []byte) error { calls++; return nil }, &calls
			},
			wantErr: false, expectedSetCalled: true, expectedSendMailCalls: 0,
		},
		{
			name:         "Storage Error",
			serverConfig: Configuration{Relays: RelayConfigurations{}},
			envelope:     smtpd.Envelope{Sender: "s@s.com", Recipients: []string{"r@r.com"}, Data: minimalEmailData},
			mockIoStoreSetup: func(ms *mockIoStorage) {
				ms.SetError = fmt.Errorf("storage set error")
			},
			smtpSendMailFnSetup: func() (func(string, smtp.Auth, string, []string, []byte) error, *int) {
				calls := 0; return func(string, smtp.Auth, string, []string, []byte) error { calls++; return nil }, &calls
			},
			wantErr: true, expectedSetCalled: true, expectedSendMailCalls: 0,
		},
		{
			name: "Successful Auto-Relay",
			serverConfig: Configuration{Relays: RelayConfigurations{
				"r1": {Enabled: true, AutoRelay: true, Addr: "relay.addr:25", Mechanism: RelayAuthModeNone},
			}},
			envelope:         smtpd.Envelope{Sender: "s@s.com", Recipients: []string{"r@r.com"}, Data: minimalEmailData},
			mockIoStoreSetup: func(ms *mockIoStorage) { ms.SetUUID = "uuid" },
			smtpSendMailFnSetup: func() (func(string, smtp.Auth, string, []string, []byte) error, *int) {
				calls := 0
				return func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
					calls++; if addr != "relay.addr:25" { t.Errorf("Expected relay addr 'relay.addr:25', got %s", addr)}; return nil
				}, &calls
			},
			wantErr: false, expectedSetCalled: true, expectedSendMailCalls: 1,
		},
		{
			name: "Malformed Email Data",
			serverConfig: Configuration{Relays: RelayConfigurations{}},
			envelope:     smtpd.Envelope{Sender: "s@s.com", Recipients: []string{"r@r.com"}, Data: []byte("Invalid Email")},
			mockIoStoreSetup: func(ms *mockIoStorage) {}, // Set should not be called
			smtpSendMailFnSetup: func() (func(string, smtp.Auth, string, []string, []byte) error, *int) {
				calls := 0; return func(string, smtp.Auth, string, []string, []byte) error { calls++; return nil }, &calls
			},
			wantErr: true, expectedSetCalled: false, expectedSendMailCalls: 0,
		},
		// Add other test cases from the original list as needed, simplified for brevity here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStore := &mockIoStorage{}
			if tt.mockIoStoreSetup != nil {
				tt.mockIoStoreSetup(mockStore)
			}
			
			s := NewServer(tt.serverConfig, mockStore)

			sendMailMock, sendMailCalls := tt.smtpSendMailFnSetup()
			smtpSendMailFn = sendMailMock

			err := s.handler(mockPeer, tt.envelope)

			if (err != nil) != tt.wantErr {
				t.Errorf("Server.handler() error = %v, wantErr %v", err, tt.wantErr)
			}
			if mockStore.SetCalled != tt.expectedSetCalled {
				t.Errorf("Server.handler() mockStore.SetCalled = %v, want %v", mockStore.SetCalled, tt.expectedSetCalled)
			}
			if *sendMailCalls != tt.expectedSendMailCalls {
				t.Errorf("Server.handler() smtpSendMailFn calls = %d, want %d", *sendMailCalls, tt.expectedSendMailCalls)
			}
		})
	}
}
