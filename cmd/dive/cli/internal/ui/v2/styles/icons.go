package styles

// --- File Icons ---

var (
	// IconDirOpen is the icon for an open directory
	IconDirOpen = "󰝰 " // nf-md-folder_open

	// IconDirClosed is the icon for a closed directory
	IconDirClosed = "󰉋 " // nf-md-folder

	// IconFile is the icon for a regular file
	IconFile = "󰈔 " // nf-md-file

	// IconSymlink is the icon for a symbolic link
	IconSymlink = "🔗 "
)

// --- Diff Type Icons ---

var (
	// IconAdded indicates an added file
	IconAdded = "✨ "

	// IconRemoved indicates a removed file
	IconRemoved = "❌ "

	// IconModified indicates a modified file
	IconModified = "✏️ "
)

// --- Action Icons ---

var (
	// IconCopy indicates copy action
	// FIX: Use nf-md-content_copy (Material Design) instead of nf-fa-copy
	// This matches IconFile/IconDir style and is better supported in NF v3+
	IconCopy = "󰆏 " // nf-md-content_copy
)
