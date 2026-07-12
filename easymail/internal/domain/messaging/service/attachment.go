package service

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"
)
func listAttachmentsFromFile(root, filePath string, mailSize int64) ([]AttachmentDTO, error) {
	full := filepath.Join(root, filePath)
	f, err := os.Open(full)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	msg, err := mail.ReadMessage(f)
	if err != nil {
		return nil, nil
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, nil
	}

	var atts []AttachmentDTO
	idx := 0
	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		cd := part.Header.Get("Content-Disposition")
		if cd == "" {
			idx++
			continue
		}
		disp, dispParams, err := mime.ParseMediaType(cd)
		if err != nil {
			idx++
			continue
		}
		if disp == "attachment" || (disp == "inline" && dispParams["filename"] != "") {
			filename := dispParams["filename"]
			if filename == "" {
				filename = dispParams["name"]
			}
			if filename == "" {
				filename = fmt.Sprintf("attachment_%d", idx)
			}
			data, _ := io.ReadAll(part)
			atts = append(atts, AttachmentDTO{
				Index:       idx,
				Name:        filename,
				Size:        int64(len(data)),
				ContentType: part.Header.Get("Content-Type"),
			})
		}
		idx++
	}
	return atts, nil
}

func openAttachmentFromFile(root, filePath string, index int) (io.ReadCloser, string, string, int64, error) {
	full := filepath.Join(root, filePath)
	f, err := os.Open(full)
	if err != nil {
		return nil, "", "", 0, err
	}
	defer f.Close()

	msg, err := mail.ReadMessage(f)
	if err != nil {
		return nil, "", "", 0, err
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, "", "", 0, fmt.Errorf("no multipart content")
	}

	curIdx := 0
	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		cd := part.Header.Get("Content-Disposition")
		if cd == "" {
			curIdx++
			continue
		}
		disp, dispParams, err := mime.ParseMediaType(cd)
		if err != nil {
			curIdx++
			continue
		}
		if disp == "attachment" || (disp == "inline" && dispParams["filename"] != "") {
			if curIdx == index {
				filename := dispParams["filename"]
				if filename == "" {
					filename = dispParams["name"]
				}
				ct := part.Header.Get("Content-Type")
				data, _ := io.ReadAll(part)
				return io.NopCloser(bytes.NewReader(data)), filename, ct, int64(len(data)), nil
			}
		}
		curIdx++
	}
	return nil, "", "", 0, fmt.Errorf("attachment index %d not found", index)
}

// decodeTransferEncoding decodes body data based on Content-Transfer-Encoding header
func decodeTransferEncoding(data []byte, encoding string) []byte {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		dst := make([]byte, base64.StdEncoding.DecodedLen(len(data)))
		n, err := base64.StdEncoding.Decode(dst, data)
		if err == nil {
			return dst[:n]
		}
		// Try with line breaks stripped
		stripped := bytes.ReplaceAll(data, []byte("\n"), nil)
		stripped = bytes.ReplaceAll(stripped, []byte("\r"), nil)
		dst2 := make([]byte, base64.StdEncoding.DecodedLen(len(stripped)))
		n2, err2 := base64.StdEncoding.Decode(dst2, stripped)
		if err2 == nil {
			return dst2[:n2]
		}
		return data
	case "quoted-printable":
		src := quotedprintable.NewReader(bytes.NewReader(data))
		dst, err := io.ReadAll(src)
		if err == nil {
			return dst
		}
		return data
	default:
		return data
	}
}

// extractBodyFromFile reads a raw .eml file and extracts the text/html body content.
func extractBodyFromFile(fullPath string) (string, error) {
	full := fullPath
	f, err := os.Open(full)
	if err != nil {
		return "", err
	}
	defer f.Close()

	msg, err := mail.ReadMessage(f)
	if err != nil {
		return "", err
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil {
		// Not multipart, read body directly
		data, _ := io.ReadAll(msg.Body)
		enc := msg.Header.Get("Content-Transfer-Encoding")
		return string(decodeTransferEncoding(data, enc)), nil
	}

	if !strings.HasPrefix(mediaType, "multipart/") {
		data, _ := io.ReadAll(msg.Body)
		enc := msg.Header.Get("Content-Transfer-Encoding")
		return string(decodeTransferEncoding(data, enc)), nil
	}

	// Multipart - extract the first text/plain or text/html part
	mr := multipart.NewReader(msg.Body, params["boundary"])
	var htmlBody string
	var textBody string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		ct, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		enc := part.Header.Get("Content-Transfer-Encoding")
		if strings.HasPrefix(ct, "multipart/") {
			// Nested multipart - recurse using part body
			data, _ := io.ReadAll(part)
			nested := extractNestedBody(decodeTransferEncoding(data, enc), part.Header.Get("Content-Type"))
			if nested != "" {
				return nested, nil
			}
			continue
		}
		data, _ := io.ReadAll(part)
		decoded := decodeTransferEncoding(data, enc)
		if strings.HasPrefix(ct, "text/html") {
			htmlBody = string(decoded)
		} else if strings.HasPrefix(ct, "text/plain") {
			textBody = string(decoded)
		}
	}
	if htmlBody != "" {
		return htmlBody, nil
	}
	return textBody, nil
}

// extractNestedBody reads a nested multipart body from raw bytes
func extractNestedBody(data []byte, contentType string) string {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	mr := multipart.NewReader(bytes.NewReader(data), params["boundary"])
	var htmlBody string
	var textBody string
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		ct, _, _ := mime.ParseMediaType(part.Header.Get("Content-Type"))
		enc := part.Header.Get("Content-Transfer-Encoding")
		partData, _ := io.ReadAll(part)
		decoded := decodeTransferEncoding(partData, enc)
		if strings.HasPrefix(ct, "text/html") {
			htmlBody = string(decoded)
		} else if strings.HasPrefix(ct, "text/plain") {
			textBody = string(decoded)
		}
	}
	if htmlBody != "" {
		return htmlBody
	}
	return textBody
}

// createAttachmentsZip opens a .eml file and creates a ZIP archive of all attachments.
func createAttachmentsZip(root, filePath string) ([]byte, string, error) {
	full := filepath.Join(root, filePath)
	f, err := os.Open(full)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()

	msg, err := mail.ReadMessage(f)
	if err != nil {
		return nil, "", err
	}

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		return nil, "", fmt.Errorf("no multipart content")
	}

	var zipBuf bytes.Buffer
	zipWriter := zip.NewWriter(&zipBuf)
	attachmentCount := 0

	mr := multipart.NewReader(msg.Body, params["boundary"])
	for {
		part, err := mr.NextPart()
		if err != nil {
			break
		}
		cd := part.Header.Get("Content-Disposition")
		if cd == "" {
			continue
		}
		disp, dispParams, err := mime.ParseMediaType(cd)
		if err != nil {
			continue
		}
		if disp == "attachment" || (disp == "inline" && dispParams["filename"] != "") {
			filename := dispParams["filename"]
			if filename == "" {
				filename = dispParams["name"]
			}
			if filename == "" {
				filename = fmt.Sprintf("attachment_%d", attachmentCount)
			}
			data, err := io.ReadAll(part)
			if err != nil {
				continue
			}
			// Decode Content-Transfer-Encoding
			enc := part.Header.Get("Content-Transfer-Encoding")
			decoded := decodeTransferEncoding(data, enc)
			fh, err := zipWriter.Create(filename)
			if err != nil {
				continue
			}
			_, _ = fh.Write(decoded)
			attachmentCount++
		}
	}

	err = zipWriter.Close()
	if err != nil {
		return nil, "", err
	}

	if attachmentCount == 0 {
		return nil, "", fmt.Errorf("no attachments found")
	}

	return zipBuf.Bytes(), fmt.Sprintf("attachments_%d.zip", time.Now().Unix()), nil
}
