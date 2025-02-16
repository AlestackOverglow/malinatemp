# MalinaTEMP

A modern desktop application for creating and managing temporary email addresses on Mail-in-a-Box servers. Built with Go and Fyne UI framework.

> **Note**: This application is specifically designed to work with [Mail-in-a-Box](https://github.com/mail-in-a-box/mailinabox.git) mail servers. It requires administrative access to your Mail-in-a-Box instance.
>
> **Repository**: [github.com/AlestackOverglow/malinatemp](https://github.com/AlestackOverglow/malinatemp.git)

## Core Features

- 🔒 Temporary email address creation
- 📨 Real-time email monitoring
- 🔔 New message notifications
- 🌓 Dark theme interface
- 🔄 Automatic mailbox refresh
- 💾 Mailbox credentials backup

## Installation

### Requirements

- Go 1.21 or later
- Git
- A configured Mail-in-a-Box server with:
  - Administrative access
  - Enabled API
  - Running IMAP service
  - Valid SSL/TLS certificates

### Building from source

1. Clone the repository:
```bash
git clone https://github.com/AlestackOverglow/malinatemp.git
cd malinatemp
```

2. Install dependencies:
```bash
go mod download
```

3. Build the application:
```bash
go build -ldflags "-H windowsgui"
```

## Usage

1. Launch the application
2. Configure your Mail-in-a-Box server settings in Settings -> MailInABox server
3. After saving settings, restart the application
4. A new temporary email address will be automatically generated
5. Copy the email address and password using the provided buttons
6. Start receiving emails in real-time

### Main Features

#### Email Management
- Create new mailboxes
- Save current mailbox to file
- Delete all emails with one click
- Delete individual emails

#### Settings
- MailInABox server configuration
- Update frequency settings
- Enable/disable notifications
- Enable/disable automatic updates

## Configuration

The application requires initial setup through the Settings menu:

- **Server Settings**
  - API URL
  - Admin credentials
  - Domain settings
  - IMAP server address

- **Update Settings**
  - Auto-update interval (5-60 seconds)
  - Notification preferences

### Error Handling

1. **Configuration Management**
   - Settings validation before saving
   - Clear error messages
   - Configuration guidance

2. **API and IMAP Errors**
   - Connection error dialogs
   - Authentication failure messages
   - Email operation error reports

## Technical Details

- Built with Go and Fyne UI framework
- Support for various email encodings
- HTML and plain text email handling
- Secure TLS connections

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Fyne](https://fyne.io/) - Cross-platform GUI framework
- [go-imap](https://github.com/emersion/go-imap) - IMAP library for Go
- [mailinabox](https://github.com/nrdcg/mailinabox) - Mail-in-a-Box API client 