package filetree

// Column width constants for metadata display
const (
	PermWidth   = 11 // "-rwxr-xr-x"
	UidGidWidth = 9  // "0:0" or "1000:1000"
	SizeWidth   = 8  // right-aligned size
	MetaGap     = "  "
)

// FormatPermissions converts os.FileMode to Unix permission string (e.g. "-rwxr-xr-x")
func FormatPermissions(mode interface{}) string {
	var m uint32
	switch v := mode.(type) {
	case uint32:
		m = v
	case int:
		m = uint32(v)
	default:
		return "----------"
	}

	// Convert to string representation
	perms := []rune("----------")

	// File type
	if m&(1<<15) != 0 { // regular file
		perms[0] = '-'
	} else if m&(1<<14) != 0 { // directory
		perms[0] = 'd'
	} else if m&(1<<12) != 0 { // symbolic link
		perms[0] = 'l'
	}

	// Owner permissions
	if m&(1<<8) != 0 {
		perms[1] = 'r'
	}
	if m&(1<<7) != 0 {
		perms[2] = 'w'
	}
	if m&(1<<6) != 0 {
		perms[3] = 'x'
	} else if m&(1<<11) != 0 { // setuid
		perms[3] = 'S'
	}

	// Group permissions
	if m&(1<<5) != 0 {
		perms[4] = 'r'
	}
	if m&(1<<4) != 0 {
		perms[5] = 'w'
	}
	if m&(1<<3) != 0 {
		perms[6] = 'x'
	} else if m&(1<<10) != 0 { // setgid
		perms[6] = 'S'
	}

	// Other permissions
	if m&(1<<2) != 0 {
		perms[7] = 'r'
	}
	if m&(1<<1) != 0 {
		perms[8] = 'w'
	}
	if m&(1<<0) != 0 {
		perms[9] = 'x'
	} else if m&(1<<9) != 0 { // sticky bit
		perms[9] = 'T'
	}

	return string(perms)
}

// FormatUidGid formats UID:GID for display, showing "-" for default root:root (0:0)
func FormatUidGid(uid, gid uint32) string {
	if uid != 0 || gid != 0 {
		return formatUint32(uid) + ":" + formatUint32(gid)
	}
	return "-"
}

// formatUint32 formats a uint32 to string
func formatUint32(v uint32) string {
	var buf [20]byte
	i := len(buf)
	n := int64(v)
	for n > 0 {
		i--
		buf[i] = '0' + byte(n%10)
		n /= 10
	}
	if i == len(buf) {
		return "0"
	}
	return string(buf[i:])
}
