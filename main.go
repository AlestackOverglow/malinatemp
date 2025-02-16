package main

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "math/rand"
    "net/url"
    "os"
    "strings"
    "time"
    "crypto/tls"
    "mime"
    "mime/multipart"
    "mime/quotedprintable"
    "net/mail"
    "bytes"
    "image/color"
    "encoding/base64"

    "github.com/emersion/go-imap"
    "github.com/emersion/go-imap/client"
    "github.com/nrdcg/mailinabox"
    "golang.org/x/net/html"
    "golang.org/x/text/encoding/charmap"
    "fyne.io/fyne/v2"
    "fyne.io/fyne/v2/app"
    "fyne.io/fyne/v2/container"
    "fyne.io/fyne/v2/widget"
    "fyne.io/fyne/v2/theme"
    "fyne.io/fyne/v2/dialog"
    "fyne.io/fyne/v2/layout"
)

type TempMailbox struct {
    Domain     string
    Username   string
    Password   string
    ImapServer string
    Client     *mailinabox.Client
}

type Email struct {
    From        string
    Subject     string
    Content     string
    HTMLContent string
    UID         uint32
}

type Settings struct {
    ApiURL        string
    AdminEmail    string
    AdminPassword string
    Domain        string
    ImapServer    string
}

// Adding retry configuration structure
type RetryConfig struct {
    MaxAttempts     int
    InitialInterval time.Duration
    MaxInterval     time.Duration
}

// Function to perform operation with retries
func withRetry(config RetryConfig, operation func() error) error {
    var lastErr error
    interval := config.InitialInterval

    for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
        if err := operation(); err != nil {
            lastErr = err
            if attempt == config.MaxAttempts {
                return fmt.Errorf("maximum attempts exceeded (%d): %w", config.MaxAttempts, err)
            }
            
            // Increase interval but not more than maximum
            if interval < config.MaxInterval {
                interval *= 2
                if interval > config.MaxInterval {
                    interval = config.MaxInterval
                }
            }
            
            time.Sleep(interval)
            continue
        }
        return nil
    }
    return lastErr
}

func (s *Settings) Validate() error {
    if s.ApiURL == "" {
        return fmt.Errorf("API URL cannot be empty")
    }
    if s.AdminEmail == "" {
        return fmt.Errorf("Admin email cannot be empty")
    }
    if s.AdminPassword == "" {
        return fmt.Errorf("Admin password cannot be empty")
    }
    if s.Domain == "" {
        return fmt.Errorf("Domain cannot be empty")
    }
    if s.ImapServer == "" {
        return fmt.Errorf("IMAP server cannot be empty")
    }
    
    // Check URL format
    if _, err := url.Parse(s.ApiURL); err != nil {
        return fmt.Errorf("invalid API URL format: %w", err)
    }
    
    // Check email format
    if !strings.Contains(s.AdminEmail, "@") {
        return fmt.Errorf("invalid admin email format")
    }
    
    return nil
}

func loadSettings() (Settings, error) {
    // Default values
    settings := Settings{
        ApiURL:        "https://your.mailinabox.domain",
        AdminEmail:    "admin@your.domain",
        AdminPassword: "your_admin_password",
        Domain:        "your.domain",
        ImapServer:    "your.imap.server:993",
    }

    // Try to load settings from file
    data, err := ioutil.ReadFile("settings.json")
    if err != nil {
        if os.IsNotExist(err) {
            return settings, fmt.Errorf("settings file not found")
        }
        return settings, fmt.Errorf("error reading settings file: %w", err)
    }

    if err := json.Unmarshal(data, &settings); err != nil {
        return settings, fmt.Errorf("error parsing settings file: %w", err)
    }

    // Validate settings
    if err := settings.Validate(); err != nil {
        return settings, fmt.Errorf("invalid settings: %w", err)
    }

    return settings, nil
}

func saveSettings(settings Settings) error {
    // Validate settings before saving
    if err := settings.Validate(); err != nil {
        return fmt.Errorf("invalid settings: %w", err)
    }

    data, err := json.MarshalIndent(settings, "", "    ")
    if err != nil {
        return fmt.Errorf("error serializing settings: %w", err)
    }

    if err := ioutil.WriteFile("settings.json", data, 0644); err != nil {
        return fmt.Errorf("error saving settings: %w", err)
    }

    return nil
}

// Function to test connection
func testConnection(settings Settings) error {
    // Check API connection
    apiClient, err := mailinabox.New(settings.ApiURL, settings.AdminEmail, settings.AdminPassword)
    if err != nil {
        return fmt.Errorf("error connecting to API: %w", err)
    }

    // Test API by creating a test user
    testEmail := fmt.Sprintf("test_%s@%s", generateRandomString(8), settings.Domain)
    testPassword := generateRandomString(16)
    _, err = apiClient.Mail.AddUser(context.Background(), testEmail, testPassword, "email")
    if err != nil {
        return fmt.Errorf("error testing API: %w", err)
    }
    // Remove test user
    _, _ = apiClient.Mail.RemoveUser(context.Background(), testEmail)

    // Check IMAP connection
    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
    }
    
    imapClient, err := client.DialTLS(settings.ImapServer, tlsConfig)
    if err != nil {
        return fmt.Errorf("error connecting to IMAP: %w", err)
    }
    defer imapClient.Logout()

    return nil
}

func NewTempMailbox(apiURL, adminEmail, adminPassword, domain, imapServer string) (*TempMailbox, error) {
    client, err := mailinabox.New(apiURL, adminEmail, adminPassword)
    if err != nil {
        return nil, fmt.Errorf("error creating client: %w", err)
    }

    return &TempMailbox{
        Domain:     domain,
        ImapServer: imapServer,
        Client:     client,
    }, nil
}

func (tm *TempMailbox) Create() error {
    tm.Username = generateRandomString(10)
    tm.Password = generateRandomString(16)

    email := fmt.Sprintf("%s@%s", tm.Username, tm.Domain)
    
    _, err := tm.Client.Mail.AddUser(context.Background(), email, tm.Password, "email")
    if err != nil {
        return fmt.Errorf("error creating user: %w", err)
    }

    return nil
}

func (tm *TempMailbox) Delete() error {
    email := fmt.Sprintf("%s@%s", tm.Username, tm.Domain)
    _, err := tm.Client.Mail.RemoveUser(context.Background(), email)
    if err != nil {
        return fmt.Errorf("error deleting user: %w", err)
    }
    return nil
}

func (tm *TempMailbox) DeleteAllMails() error {
    retryConfig := RetryConfig{
        MaxAttempts:     3,
        InitialInterval: 1 * time.Second,
        MaxInterval:     5 * time.Second,
    }
    
    return withRetry(retryConfig, func() error {
        return tm.deleteAllMailsInternal()
    })
}

func (tm *TempMailbox) deleteAllMailsInternal() error {
    email := fmt.Sprintf("%s@%s", tm.Username, tm.Domain)
    log.Printf("Deleting all mails for %s\n", email)
    
    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
    }
    
    imapClient, err := client.DialTLS(tm.ImapServer, tlsConfig)
    if err != nil {
        return fmt.Errorf("error connecting to IMAP: %w", err)
    }
    defer imapClient.Logout()

    if err := imapClient.Login(email, tm.Password); err != nil {
        return fmt.Errorf("error authenticating IMAP: %w", err)
    }

    // Select INBOX
    mbox, err := imapClient.Select("INBOX", false)
    if err != nil {
        return fmt.Errorf("error selecting folder: %w", err)
    }

    if mbox.Messages == 0 {
        return nil
    }

    // Create set for all messages
    seqSet := new(imap.SeqSet)
    seqSet.AddRange(1, mbox.Messages)

    // Mark all messages as deleted
    item := imap.FormatFlagsOp(imap.AddFlags, true)
    flags := []interface{}{imap.DeletedFlag}
    if err := imapClient.Store(seqSet, item, flags, nil); err != nil {
        return fmt.Errorf("error marking mails for deletion: %w", err)
    }

    // Physically delete marked messages
    if err := imapClient.Expunge(nil); err != nil {
        return fmt.Errorf("error deleting mails: %w", err)
    }

    return nil
}

func extractTextFromHTML(htmlContent string) string {
    doc, err := html.Parse(strings.NewReader(htmlContent))
    if err != nil {
        return htmlContent
    }

    var text strings.Builder
    var extract func(*html.Node)
    extract = func(n *html.Node) {
        if n.Type == html.TextNode {
            text.WriteString(strings.TrimSpace(n.Data))
            if len(n.Data) > 0 {
                text.WriteString("\n")
            }
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            extract(c)
        }
    }
    extract(doc)
    return strings.TrimSpace(text.String())
}

func getTextFromPart(part *multipart.Part) (string, error) {
    contentType := part.Header.Get("Content-Type")
    mediaType, _, err := mime.ParseMediaType(contentType)
    if err != nil {
        return "", err
    }

    content, err := ioutil.ReadAll(part)
    if err != nil {
        return "", err
    }

    if strings.HasPrefix(mediaType, "text/html") {
        return extractTextFromHTML(string(content)), nil
    }
    return string(content), nil
}

func decodeRFC2047(s string) string {
    decoded, err := (&mime.WordDecoder{}).DecodeHeader(s)
    if err != nil {
        return s
    }
    return decoded
}

func decodeContent(content []byte, encoding string) ([]byte, error) {
    encoding = strings.ToLower(encoding)
    switch encoding {
    case "base64":
        decoded := make([]byte, base64.StdEncoding.DecodedLen(len(content)))
        n, err := base64.StdEncoding.Decode(decoded, content)
        if err != nil {
            return content, err
        }
        return decoded[:n], nil
    case "quoted-printable":
        reader := quotedprintable.NewReader(bytes.NewReader(content))
        decoded, err := ioutil.ReadAll(reader)
        if err != nil {
            return content, err
        }
        return decoded, nil
    default:
        return content, nil
    }
}

// Updated CheckMail method with retry support
func (tm *TempMailbox) CheckMail() ([]Email, error) {
    var emails []Email
    
    retryConfig := RetryConfig{
        MaxAttempts:     3,
        InitialInterval: 1 * time.Second,
        MaxInterval:     5 * time.Second,
    }
    
    err := withRetry(retryConfig, func() error {
        var checkErr error
        emails, checkErr = tm.checkMailInternal()
        return checkErr
    })
    
    return emails, err
}

// Renamed original CheckMail method to checkMailInternal
func (tm *TempMailbox) checkMailInternal() ([]Email, error) {
    email := fmt.Sprintf("%s@%s", tm.Username, tm.Domain)
    log.Printf("Checking mail for %s\n", email)
    
    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
    }
    
    imapClient, err := client.DialTLS(tm.ImapServer, tlsConfig)
    if err != nil {
        return nil, fmt.Errorf("error connecting to IMAP: %w", err)
    }
    defer imapClient.Logout()

    if err := imapClient.Login(email, tm.Password); err != nil {
        return nil, fmt.Errorf("error authenticating IMAP: %w", err)
    }
    log.Printf("Successfully connected to IMAP\n")

    mbox, err := imapClient.Select("INBOX", false)
    if err != nil {
        return nil, fmt.Errorf("error selecting folder: %w", err)
    }
    log.Printf("Selected INBOX, mails: %d\n", mbox.Messages)

    if mbox.Messages == 0 {
        return []Email{}, nil
    }

    seqSet := new(imap.SeqSet)
    seqSet.AddRange(1, mbox.Messages)

    messages := make(chan *imap.Message, 10)
    done := make(chan error, 1)

    // Request all message data
    items := []imap.FetchItem{imap.FetchEnvelope, imap.FetchBody, imap.FetchBodyStructure, "BODY[]"}

    go func() {
        done <- imapClient.Fetch(seqSet, items, messages)
    }()

    var emails []Email
    for msg := range messages {
        email := Email{
            Subject: decodeRFC2047(msg.Envelope.Subject),
            UID:     msg.Uid,
        }
        
        if len(msg.Envelope.From) > 0 {
            addr := msg.Envelope.From[0]
            if addr.PersonalName != "" {
                email.From = decodeRFC2047(addr.PersonalName)
            } else {
                email.From = fmt.Sprintf("%s@%s", addr.MailboxName, addr.HostName)
            }
        }

        log.Printf("Processing mail from %s with subject %s\n", email.From, email.Subject)

        // Get message body
        for _, literal := range msg.Body {
            buf := new(bytes.Buffer)
            _, err := io.Copy(buf, literal)
            if err != nil {
                log.Printf("Error reading message body: %v\n", err)
                continue
            }

            // Try to read as MIME message
            m, err := mail.ReadMessage(bytes.NewReader(buf.Bytes()))
            if err != nil {
                log.Printf("Error parsing MIME: %v\n", err)
                // Try to decode as plain text
                decoded, err := decodeCharset(buf.Bytes(), "")
                if err == nil {
                    email.Content = decoded
                } else {
                    email.Content = decodeRFC2047(buf.String())
                }
                continue
            }

            mediaType, params, err := mime.ParseMediaType(m.Header.Get("Content-Type"))
            if err != nil {
                log.Printf("Error determining content type: %v\n", err)
                decoded, err := decodeCharset(buf.Bytes(), "")
                if err == nil {
                    email.Content = decoded
                } else {
                    email.Content = decodeRFC2047(buf.String())
                }
                continue
            }

            log.Printf("Content type: %s\n", mediaType)

            if strings.HasPrefix(mediaType, "multipart/") {
                mr := multipart.NewReader(m.Body, params["boundary"])
                
                // Process only text parts
                for {
                    part, err := mr.NextPart()
                    if err == io.EOF {
                        break
                    }
                    if err != nil {
                        log.Printf("Error reading part: %v\n", err)
                        continue
                    }

                    partType, partParams, err := mime.ParseMediaType(part.Header.Get("Content-Type"))
                    if err != nil {
                        continue
                    }
                    
                    partCharset := partParams["charset"]
                    if partCharset == "" {
                        partCharset = "utf-8"
                    }

                    body, err := ioutil.ReadAll(part)
                    if err != nil {
                        continue
                    }

                    decodedBody, err := decodeContent(body, part.Header.Get("Content-Transfer-Encoding"))
                    if err != nil {
                        decodedBody = body
                    }

                    if strings.HasPrefix(partType, "text/plain") {
                        decoded, err := decodeCharset(decodedBody, partCharset)
                        if err == nil {
                            if email.Content == "" {
                                email.Content = decoded
                            } else {
                                email.Content += "\n\n" + decoded
                            }
                        } else {
                            if email.Content == "" {
                                email.Content = string(decodedBody)
                            } else {
                                email.Content += "\n\n" + string(decodedBody)
                            }
                        }
                        log.Printf("Added message text\n")
                    } else if strings.HasPrefix(partType, "text/html") {
                        decoded, err := decodeCharset(decodedBody, partCharset)
                        if err == nil {
                            email.HTMLContent = decoded
                            if email.Content == "" {
                                email.Content = extractTextFromHTML(decoded)
                            }
                        } else {
                            email.HTMLContent = string(decodedBody)
                            if email.Content == "" {
                                email.Content = extractTextFromHTML(string(decodedBody))
                            }
                        }
                        log.Printf("Added HTML message text\n")
                    }
                }
            } else if strings.HasPrefix(mediaType, "text/plain") {
                body, _ := ioutil.ReadAll(m.Body)
                decodedBody, err := decodeContent(body, m.Header.Get("Content-Transfer-Encoding"))
                if err != nil {
                    decodedBody = body
                }
                decoded, err := decodeCharset(decodedBody, params["charset"])
                if err == nil {
                    email.Content = decoded
                } else {
                    email.Content = string(decodedBody)
                }
            } else if strings.HasPrefix(mediaType, "text/html") {
                body, _ := ioutil.ReadAll(m.Body)
                decodedBody, err := decodeContent(body, m.Header.Get("Content-Transfer-Encoding"))
                if err != nil {
                    log.Printf("Error decoding content: %v\n", err)
                    decodedBody = body
                }
                decoded, err := decodeCharset(decodedBody, params["charset"])
                if err == nil {
                    email.HTMLContent = decoded
                    email.Content = extractTextFromHTML(decoded)
                } else {
                    email.HTMLContent = string(decodedBody)
                    email.Content = extractTextFromHTML(string(decodedBody))
                }
            }

            if email.Content != "" {
                // Clear content from null bytes and extra spaces
                email.Content = strings.TrimSpace(strings.ReplaceAll(email.Content, "\x00", ""))
                // Add logging for debugging
                log.Printf("Message content after processing: %s\n", email.Content)
            }
        }

        log.Printf("Adding message to list\n")
        emails = append(emails, email)
    }

    if err := <-done; err != nil {
        return nil, fmt.Errorf("error getting messages: %w", err)
    }

    // Sort messages in reverse order (newest on top)
    for i := len(emails)/2 - 1; i >= 0; i-- {
        opp := len(emails) - 1 - i
        emails[i], emails[opp] = emails[opp], emails[i]
    }

    log.Printf("Total processed messages: %d\n", len(emails))
    return emails, nil
}

func generateRandomString(length int) string {
    const charset = "abcdefghijklmnopqrstuvwxyz" // Only small English letters
    seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))
    
    b := strings.Builder{}
    b.Grow(length)
    for i := 0; i < length; i++ {
        b.WriteByte(charset[seededRand.Intn(len(charset))])
    }
    return b.String()
}

func decodeCharset(content []byte, charset string) (string, error) {
    charset = strings.ToLower(charset)
    switch charset {
    case "utf-8", "us-ascii":
        return string(content), nil
    case "koi8-r":
        decoder := charmap.KOI8R.NewDecoder()
        decoded, err := decoder.Bytes(content)
        if err != nil {
            return "", err
        }
        return string(decoded), nil
    case "windows-1251", "cp1251":
        decoder := charmap.Windows1251.NewDecoder()
        decoded, err := decoder.Bytes(content)
        if err != nil {
            return "", err
        }
        return string(decoded), nil
    case "iso-8859-5":
        decoder := charmap.ISO8859_5.NewDecoder()
        decoded, err := decoder.Bytes(content)
        if err != nil {
            return "", err
        }
        return string(decoded), nil
    default:
        // Try to guess encoding
        // First try windows-1251 as the most common
        decoder := charmap.Windows1251.NewDecoder()
        decoded, err := decoder.Bytes(content)
        if err == nil && !strings.Contains(string(decoded), "") {
            return string(decoded), nil
        }
        
        // Then try KOI8-R
        decoder = charmap.KOI8R.NewDecoder()
        decoded, err = decoder.Bytes(content)
        if err == nil && !strings.Contains(string(decoded), "") {
            return string(decoded), nil
        }

        return string(content), fmt.Errorf("unsupported encoding: %s", charset)
    }
}

// Create custom theme
type customTheme struct {
    fyne.Theme
}

func newCustomTheme() fyne.Theme {
    return &customTheme{theme.DefaultTheme()}
}

func (t *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
    switch name {
    case theme.ColorNameBackground:
        return color.NRGBA{R: 33, G: 33, B: 33, A: 255} // Dark gray background
    case theme.ColorNameForeground:
        return color.NRGBA{R: 255, G: 255, B: 255, A: 255} // White text
    case theme.ColorNameInputBackground:
        return color.NRGBA{R: 45, G: 45, B: 45, A: 255} // Slightly lighter than background for input fields
    }
    return t.Theme.Color(name, variant)
}

func saveMailboxToFile(email, password string) error {
    file, err := os.OpenFile("saved_mailboxes.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    timestamp := time.Now().Format("2006-01-02 15:04:05")
    _, err = fmt.Fprintf(file, "[%s] Email: %s | Password: %s\n", timestamp, email, password)
    return err
}

// Updated DeleteMail method with retry support
func (tm *TempMailbox) DeleteMail(uid uint32) error {
    retryConfig := RetryConfig{
        MaxAttempts:     3,
        InitialInterval: 1 * time.Second,
        MaxInterval:     5 * time.Second,
    }
    
    return withRetry(retryConfig, func() error {
        return tm.deleteMailInternal(uid)
    })
}

// Renamed original DeleteMail method to deleteMailInternal
func (tm *TempMailbox) deleteMailInternal(uid uint32) error {
    email := fmt.Sprintf("%s@%s", tm.Username, tm.Domain)
    log.Printf("Deleting mail with UID %d for %s\n", uid, email)
    
    tlsConfig := &tls.Config{
        InsecureSkipVerify: true,
    }
    
    imapClient, err := client.DialTLS(tm.ImapServer, tlsConfig)
    if err != nil {
        return fmt.Errorf("error connecting to IMAP: %w", err)
    }
    defer imapClient.Logout()

    if err := imapClient.Login(email, tm.Password); err != nil {
        return fmt.Errorf("error authenticating IMAP: %w", err)
    }

    // Select INBOX
    _, err = imapClient.Select("INBOX", false)
    if err != nil {
        return fmt.Errorf("error selecting folder: %w", err)
    }

    // Create set for message by UID
    seqSet := new(imap.SeqSet)
    seqSet.AddNum(uid)

    // Mark message as deleted
    item := imap.FormatFlagsOp(imap.AddFlags, true)
    flags := []interface{}{imap.DeletedFlag}
    if err := imapClient.UidStore(seqSet, item, flags, nil); err != nil {
        return fmt.Errorf("error marking message for deletion: %w", err)
    }

    // Physically delete marked message
    if err := imapClient.Expunge(nil); err != nil {
        return fmt.Errorf("error deleting message: %w", err)
    }

    return nil
}

func showSettingsDialog(window fyne.Window, settings Settings, onSave func(Settings)) {
    // Create input fields
    apiURLEntry := widget.NewEntry()
    apiURLEntry.SetText(settings.ApiURL)
    
    adminEmailEntry := widget.NewEntry()
    adminEmailEntry.SetText(settings.AdminEmail)
    
    adminPasswordEntry := widget.NewEntry()
    adminPasswordEntry.SetText(settings.AdminPassword)
    
    domainEntry := widget.NewEntry()
    domainEntry.SetText(settings.Domain)
    
    imapServerEntry := widget.NewEntry()
    imapServerEntry.SetText(settings.ImapServer)

    // Create progress indicator
    progress := widget.NewProgressBarInfinite()
    progress.Hide()

    // Create test connection button
    testButton := widget.NewButton("Test connection", func() {
        progress.Show()
        newSettings := Settings{
            ApiURL:        apiURLEntry.Text,
            AdminEmail:    adminEmailEntry.Text,
            AdminPassword: adminPasswordEntry.Text,
            Domain:        domainEntry.Text,
            ImapServer:    imapServerEntry.Text,
        }

        // Validate settings in separate goroutine
        go func() {
            if err := testConnection(newSettings); err != nil {
                // Return to main goroutine for UI update
                window.Canvas().Refresh(progress)
                progress.Hide()
                dialog.ShowError(err, window)
                return
            }
            
            window.Canvas().Refresh(progress)
            progress.Hide()
            dialog.ShowInformation("Success", "Connection established", window)
        }()
    })

    // Create form
    formContent := container.NewVBox(
        container.NewHBox(widget.NewLabel("API URL:"), layout.NewSpacer()),
        container.NewMax(apiURLEntry),
        container.NewHBox(widget.NewLabel("Admin email:"), layout.NewSpacer()),
        container.NewMax(adminEmailEntry),
        container.NewHBox(widget.NewLabel("Admin password:"), layout.NewSpacer()),
        container.NewMax(adminPasswordEntry),
        container.NewHBox(widget.NewLabel("Domain:"), layout.NewSpacer()),
        container.NewMax(domainEntry),
        container.NewHBox(widget.NewLabel("IMAP server:"), layout.NewSpacer()),
        container.NewMax(imapServerEntry),
        progress,
        container.NewHBox(
            testButton,
            layout.NewSpacer(),
            widget.NewButton("Save", func() {
                progress.Show()
                
                newSettings := Settings{
                    ApiURL:        apiURLEntry.Text,
                    AdminEmail:    adminEmailEntry.Text,
                    AdminPassword: adminPasswordEntry.Text,
                    Domain:        domainEntry.Text,
                    ImapServer:    imapServerEntry.Text,
                }
                
                // Validate settings
                if err := newSettings.Validate(); err != nil {
                    progress.Hide()
                    dialog.ShowError(err, window)
                    return
                }
                
                // Try to save
                if err := saveSettings(newSettings); err != nil {
                    progress.Hide()
                    dialog.ShowError(err, window)
                    return
                }
                
                progress.Hide()
                onSave(newSettings)
                dialog.ShowInformation("Success", "Settings saved", window)
            }),
        ),
    )

    // Create dialog with increased size
    settingsDialog := dialog.NewCustom("Settings", "Close", container.NewPadded(formContent), window)
    settingsDialog.Resize(fyne.NewSize(400, 400))
    settingsDialog.Show()
}

func main() {
    // Load settings
    settings, err := loadSettings()
    
    // Create application
    myApp := app.NewWithID("com.tempmail.app")
    myApp.SetIcon(theme.MailComposeIcon())
    myApp.Settings().SetTheme(newCustomTheme())
    
    // Set path to PowerShell for notifications
    os.Setenv("PATH", os.Getenv("PATH")+";C:\\Windows\\System32\\WindowsPowerShell\\v1.0")
    
    window := myApp.NewWindow("Temporary email mailbox")

    // Function to show settings-only interface
    showSettingsInterface := func() {
        // Show information dialog
        dialog.ShowInformation(
            "Configuration Required",
            "Please configure the application by going to Settings -> MailInABox server and entering your server details.",
            window,
        )

        // Create main menu with only settings
        mainMenu := fyne.NewMainMenu(
            fyne.NewMenu("Settings",
                fyne.NewMenuItem("MailInABox server", func() {
                    showSettingsDialog(window, settings, func(newSettings Settings) {
                        settings = newSettings
                        // After saving settings, restart the application
                        dialog.ShowInformation(
                            "Settings Saved",
                            "Settings have been saved. Please restart the application.",
                            window,
                        )
                    })
                }),
            ),
        )

        window.SetMainMenu(mainMenu)
        window.Resize(fyne.NewSize(500, 600))
        window.CenterOnScreen()
        window.ShowAndRun()
    }

    // Handle settings and initialization errors
    if err != nil {
        showSettingsInterface()
        return
    }

    // Try to create temporary mailbox
    mailbox, err := NewTempMailbox(
        settings.ApiURL,
        settings.AdminEmail,
        settings.AdminPassword,
        settings.Domain,
        settings.ImapServer,
    )
    if err != nil {
        log.Printf("Error creating temporary mailbox: %v\n", err)
        showSettingsInterface()
        return
    }

    if err := mailbox.Create(); err != nil {
        log.Printf("Error creating mailbox: %v\n", err)
        showSettingsInterface()
        return
    }

    // Continue with normal application initialization
    // Create loading indicator
    progress := widget.NewProgressBarInfinite()
    progress.Hide()

    // Create fields for displaying mailbox information
    emailEntry := widget.NewEntry()
    emailEntry.SetText(fmt.Sprintf("%s@%s", mailbox.Username, mailbox.Domain))
    emailEntry.Disable()
    emailEntry.Resize(fyne.NewSize(200, 36))

    passwordEntry := widget.NewEntry()
    passwordEntry.SetText(mailbox.Password)
    passwordEntry.Disable()
    passwordEntry.Resize(fyne.NewSize(200, 36))

    // Create copy buttons
    copyEmailBtn := widget.NewButton("Copy Email", func() {
        window.Clipboard().SetContent(emailEntry.Text)
    })

    copyPassBtn := widget.NewButton("Copy Password", func() {
        window.Clipboard().SetContent(passwordEntry.Text)
    })

    // Create copy containers with copy buttons
    emailBox := container.NewHBox(
        container.NewGridWrap(fyne.NewSize(200, 36), emailEntry),
        copyEmailBtn,
    )
    passwordBox := container.NewHBox(
        container.NewGridWrap(fyne.NewSize(200, 36), passwordEntry),
        copyPassBtn,
    )

    // Create check box for automatic update
    autoUpdateCheck := widget.NewCheck("Automatic update", nil)
    autoUpdateCheck.SetChecked(true)

    // Create check box for notifications
    notificationsCheck := widget.NewCheck("Notifications", nil)
    notificationsCheck.SetChecked(true)

    // Create slider for update period (5 to 60 seconds)
    updatePeriodSlider := widget.NewSlider(5, 60)
    updatePeriodSlider.SetValue(5)
    updatePeriodLabel := widget.NewLabel("Update period: 5 sec")
    updatePeriodSlider.OnChanged = func(value float64) {
        updatePeriodLabel.SetText(fmt.Sprintf("Update period: %.0f sec", value))
    }

    // Create manual update button
    updateButton := widget.NewButton("Update", nil)
    updateButton.Disable() // Initially disabled, as automatic update is enabled

    // Create container for update management
    container.NewVBox(
        container.NewHBox(
            autoUpdateCheck,
            updateButton,
        ),
        container.NewHBox(
            notificationsCheck,
        ),
        updatePeriodLabel,
        updatePeriodSlider,
    )

    // Create list for displaying messages
    var emails []Email
    
    // Use VBox instead of GridWrap for better adaptability
    emailsList := container.NewVBox()

    // Create delete all button
    deleteAllButton := widget.NewButton("Delete all mails", func() {
        progress.Show()
        if err := mailbox.DeleteAllMails(); err != nil {
            log.Printf("Error deleting mails: %v\n", err)
            dialog.ShowError(fmt.Errorf("Error deleting mails: %v", err), window)
        } else {
            // Clear message list in interface
            emails = []Email{}
            emailsList.Objects = nil
            emailsList.Refresh()
        }
        progress.Hide()
    })

    // Messages update function
    var updateEmailsList func([]Email)
    updateEmailsList = func(newEmails []Email) {
        emailsList.Objects = nil // Clear list
        
        for _, email := range newEmails {
            email := email // Create new variable for closure
            
            // Create labels for headers
            fromLabel := widget.NewLabelWithStyle(
                "From: "+email.From,
                fyne.TextAlignLeading,
                fyne.TextStyle{Bold: true},
            )
            fromLabel.Wrapping = fyne.TextWrapWord
            
            subjectLabel := widget.NewLabelWithStyle(
                "Subject: "+email.Subject,
                fyne.TextAlignLeading,
                fyne.TextStyle{Bold: true},
            )
            subjectLabel.Wrapping = fyne.TextWrapWord

            // Create delete button
            deleteBtn := widget.NewButton("Delete", func() {
                progress.Show()
                if err := mailbox.DeleteMail(email.UID); err != nil {
                    log.Printf("Error deleting message: %v\n", err)
                    dialog.ShowError(fmt.Errorf("Error deleting message: %v", err), window)
                    progress.Hide()
                    return
                }
                // Get new message list
                newEmails, err := mailbox.CheckMail()
                if err != nil {
                    log.Printf("Error updating message list: %v\n", err)
                    dialog.ShowError(fmt.Errorf("Error updating message list: %v", err), window)
                    progress.Hide()
                    return
                }
                emails = newEmails
                updateEmailsList(emails)
                progress.Hide()
            })

            // Create switch between HTML and text representation
            var content *widget.Entry
            var htmlView *widget.RichText

            content = widget.NewMultiLineEntry()
            content.SetText(email.Content)
            content.Disable()
            content.Wrapping = fyne.TextWrapWord
            content.TextStyle = fyne.TextStyle{Bold: true}
            content.SetMinRowsVisible(8)

            htmlView = widget.NewRichTextFromMarkdown(email.HTMLContent)
            htmlView.Wrapping = fyne.TextWrapWord
            htmlView.Hide()

            viewTypeBtn := widget.NewButton("Switch view", func() {
                if content.Visible() {
                    content.Hide()
                    htmlView.Show()
                } else {
                    htmlView.Hide()
                    content.Show()
                }
            })

            // Create content container
            contentBox := container.NewVBox(
                content,
                htmlView,
            )

            // Create card for message with adaptive size
            card := widget.NewCard(
                "",
                "",
                container.NewVBox(
                    container.NewPadded(
                        container.NewVBox(
                            fromLabel,
                            subjectLabel,
                            widget.NewSeparator(),
                            contentBox,
                            container.NewHBox(
                                viewTypeBtn,
                                layout.NewSpacer(),
                                deleteBtn,
                            ),
                        ),
                    ),
                ),
            )
            
            emailsList.Add(container.NewPadded(card))
        }
        emailsList.Refresh()
    }

    // Messages update function
    updateEmails := func() {
        progress.Show()
        newEmails, err := mailbox.CheckMail()
        if err != nil {
            log.Printf("Error checking mail: %v\n", err)
            progress.Hide()
            return
        }

        log.Printf("Found messages: %d\n", len(newEmails))

        if len(newEmails) > 0 {
            // Check if there are new messages
            if len(newEmails) > len(emails) && notificationsCheck.Checked {
                // Get new message count
                newCount := len(newEmails) - len(emails)
                // Send notification
                notification := fyne.NewNotification(
                    "New messages",
                    fmt.Sprintf("Received %d new messages", newCount),
                )
                myApp.SendNotification(notification)
                log.Printf("Sent notification about %d new messages\n", newCount)
            }
            
            emails = newEmails
            window.Canvas().Refresh(emailsList)
            updateEmailsList(emails)
        }
        
        progress.Hide()
    }

    // Set handlers
    updateButton.OnTapped = updateEmails
    autoUpdateCheck.OnChanged = func(checked bool) {
        updateButton.Disable()
        if !checked {
            updateButton.Enable()
        }
    }

    // Create container for mailbox information
    infoBox := container.NewVBox(
        widget.NewLabelWithStyle("Email:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        container.NewHBox(
            container.NewMax(emailBox),
            layout.NewSpacer(),
        ),
        widget.NewLabelWithStyle("Password:", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
        container.NewHBox(
            container.NewMax(passwordBox),
            layout.NewSpacer(),
        ),
        widget.NewSeparator(),
        container.NewHBox(
            deleteAllButton,
            layout.NewSpacer(),
            updateButton,
        ),
        progress,
    )

    // Create scrollable container for messages with adaptive size
    scrollContainer := container.NewScroll(container.NewPadded(emailsList))
    
    // Create main container with adaptive layout
    content := container.NewBorder(
        infoBox,
        nil,
        nil,
        nil,
        scrollContainer,
    )

    // Create main menu
    mainMenu := fyne.NewMainMenu(
        fyne.NewMenu("File",
            fyne.NewMenuItem("Create new mailbox", func() {
                progress.Show()
                if err := mailbox.Delete(); err != nil {
                    log.Printf("Error deleting mailbox: %v\n", err)
                }
                if err := mailbox.Create(); err != nil {
                    log.Printf("Error creating new mailbox: %v\n", err)
                    progress.Hide()
                    return
                }
                emails = []Email{}
                emailsList.Objects = nil
                emailsList.Refresh()
                emailEntry.SetText(fmt.Sprintf("%s@%s", mailbox.Username, mailbox.Domain))
                passwordEntry.SetText(mailbox.Password)
                progress.Hide()
            }),
            fyne.NewMenuItem("Create additional mailbox", func() {
                progress.Show()
                currentEmail := fmt.Sprintf("%s@%s", mailbox.Username, mailbox.Domain)
                if err := saveMailboxToFile(currentEmail, mailbox.Password); err != nil {
                    log.Printf("Error saving mailbox: %v\n", err)
                    dialog.ShowError(fmt.Errorf("Error saving mailbox: %v", err), window)
                    progress.Hide()
                    return
                }
                if err := mailbox.Create(); err != nil {
                    log.Printf("Error creating new mailbox: %v\n", err)
                    dialog.ShowError(fmt.Errorf("Error creating new mailbox: %v", err), window)
                    progress.Hide()
                    return
                }
                emails = []Email{}
                emailsList.Objects = nil
                emailsList.Refresh()
                emailEntry.SetText(fmt.Sprintf("%s@%s", mailbox.Username, mailbox.Domain))
                passwordEntry.SetText(mailbox.Password)
                dialog.ShowInformation("Success", "Previous mailbox saved to saved_mailboxes.txt", window)
                progress.Hide()
            }),
        ),
        fyne.NewMenu("Settings",
            fyne.NewMenuItem("MailInABox server", func() {
                showSettingsDialog(window, settings, func(newSettings Settings) {
                    settings = newSettings
                    if err := mailbox.Delete(); err != nil {
                        log.Printf("Error deleting mailbox: %v\n", err)
                    }
                    newMailbox, err := NewTempMailbox(
                        settings.ApiURL,
                        settings.AdminEmail,
                        settings.AdminPassword,
                        settings.Domain,
                        settings.ImapServer,
                    )
                    if err != nil {
                        dialog.ShowError(fmt.Errorf("Error creating mailbox: %v", err), window)
                        return
                    }
                    if err := newMailbox.Create(); err != nil {
                        dialog.ShowError(fmt.Errorf("Error creating mailbox: %v", err), window)
                        return
                    }
                    mailbox = newMailbox
                    emails = []Email{}
                    emailsList.Objects = nil
                    emailsList.Refresh()
                    emailEntry.SetText(fmt.Sprintf("%s@%s", mailbox.Username, mailbox.Domain))
                    passwordEntry.SetText(mailbox.Password)
                })
            }),
            fyne.NewMenuItem("Update and notifications", func() {
                // Create update settings dialog
                updateSettingsContent := container.NewVBox(
                    autoUpdateCheck,
                    notificationsCheck,
                    updatePeriodLabel,
                    updatePeriodSlider,
                )
                updateDialog := dialog.NewCustom(
                    "Update settings",
                    "Close",
                    container.NewPadded(updateSettingsContent),
                    window,
                )
                updateDialog.Resize(fyne.NewSize(300, 200))
                updateDialog.Show()
            }),
        ),
    )

    window.SetMainMenu(mainMenu)
    window.SetContent(content)
    window.Resize(fyne.NewSize(500, 600))
    window.CenterOnScreen()

    // Start mail checking in background mode
    go func() {
        time.Sleep(2 * time.Second)

        for {
            if autoUpdateCheck.Checked {
                updateEmails()
            }
            time.Sleep(time.Duration(updatePeriodSlider.Value) * time.Second)
        }
    }()

    // Create file for logs
    logFile, err := os.OpenFile("tempmail.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err == nil {
        log.SetOutput(logFile)
    }

    // Set window close interceptor
    window.SetCloseIntercept(func() {
        dialog.ShowConfirm(
            "Confirmation",
            "Do you want to delete the current mailbox?\nClick 'Yes' to delete or 'No' to save.",
            func(delete bool) {
                if delete {
                    if err := mailbox.Delete(); err != nil {
                        log.Printf("Error deleting mailbox: %v\n", err)
                    }
                } else {
                    currentEmail := fmt.Sprintf("%s@%s", mailbox.Username, mailbox.Domain)
                    if err := saveMailboxToFile(currentEmail, mailbox.Password); err != nil {
                        log.Printf("Error saving mailbox: %v\n", err)
                    }
                }
                window.Close()
            },
            window,
        )
    })

    // Show window and start main loop
    window.ShowAndRun()
} 