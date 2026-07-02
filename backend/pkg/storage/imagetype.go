package storage

// MaxImageBytes caps every user-supplied image upload (menu photos, avatars,
// KYC documents, incident photos). Larger files must be rejected at the
// handler before any bytes reach object storage.
const MaxImageBytes = 5 << 20 // 5 MB

// AllowedImageType reports whether a multipart Content-Type is an image
// format the platform accepts. Anything else (SVG, PDF, octet-stream…) is
// rejected — user uploads are served back to browsers, so permissive types
// would enable stored-XSS / content-sniffing attacks.
func AllowedImageType(contentType string) bool {
	switch contentType {
	case "image/jpeg", "image/png", "image/webp":
		return true
	}
	return false
}
