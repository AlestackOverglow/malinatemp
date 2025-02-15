# MalinaTEMP

A modern desktop application for creating and managing temporary email addresses on Mail-in-a-Box servers. Built with Go and Fyne UI framework, this client provides a user-friendly interface for temporary email management on your self-hosted Mail-in-a-Box server.

The application includes error handling for common issues:
- Shows error messages through dialog windows
- Handles connection problems with Mail-in-a-Box API
- Manages IMAP authentication errors
- Provides feedback for email operations
- Saves settings in a configuration file

> **Note**: This application is specifically designed to work with [Mail-in-a-Box](https://mailinabox.email/) mail servers. It requires administrative access to your Mail-in-a-Box instance.
>
> **Repository**: [github.com/AlestackOverglow/malinatemp](https://github.com/AlestackOverglow/malinatemp.git)
>
> **Mail-in-a-Box**: This client works with [Mail-in-a-Box](https://github.com/mail-in-a-box/mailinabox.git) - an open source mail server solution that helps individuals take back control of their email by providing a one-click, easy-to-deploy SMTP+everything else server.

![Application Screenshot](screenshot.png)

## Features

- 🔒 Secure temporary email creation
- 📨 Real-time email monitoring
- 🔔 Desktop notifications for new messages
- 📎 Support for email attachments
- 🌓 Dark theme interface
- 🔄 Automatic mailbox refresh
- 💾 Mailbox backup and restore
- ⚙️ Configurable server settings

## Installation

### Prerequisites

- Go 1.21 or later
- Git
- A running Mail-in-a-Box server with:
  - Administrative access
  - API access enabled
  - IMAP service running
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
- Individual email deletion

#### Settings
- Configure MailInABox server settings
- Adjust update frequency
- Enable/disable notifications
- Toggle automatic updates

#### Security
- Secure connection to IMAP server
- Automatic mailbox deletion on exit (with confirmation)
- Option to save mailbox credentials before closing

## Configuration

The application requires initial configuration through the Settings menu:

- **Server Settings**
  - API URL
  - Admin credentials
  - Domain settings
  - IMAP server address

- **Update Settings**
  - Auto-update interval (5-60 seconds)
  - Notification preferences
  - Update frequency

### Error Handling

The application handles various error scenarios:

1. **Configuration Management**
   - Validates settings before saving
   - Shows clear error messages for invalid settings
   - Provides configuration guidance

2. **API and IMAP Errors**
   - Shows error dialogs for connection problems
   - Displays authentication failure messages
   - Reports email operation errors
   - Provides feedback for mailbox creation/deletion

3. **Email Operations**
   - Handles email deletion errors
   - Shows attachment saving errors
   - Reports email checking failures
   - Manages mailbox backup errors

4. **User Interface**
   - Disables buttons during operations
   - Shows progress indicator for long operations
   - Provides clear error messages in dialogs
   - Allows retrying failed operations

## Technical Details

- Built with Go and Fyne UI framework
- Supports multiple email encodings
- Handles HTML and plain text emails
- Manages email attachments
- Uses secure TLS connections

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Fyne](https://fyne.io/) - Cross-platform GUI framework
- [go-imap](https://github.com/emersion/go-imap) - IMAP library for Go
- [mailinabox](https://github.com/nrdcg/mailinabox) - Mail-in-a-Box API client 