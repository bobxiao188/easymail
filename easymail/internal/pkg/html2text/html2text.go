/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * License: AGPLv3
 */

package feature

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Html2Text extracts plain-text fragments and link URLs from HTML (best-effort, tolerant).
type Html2Text struct {
	htmlTagMapper map[string]struct{}
}

// NewHtml2Text builds a converter; tags selects allowed element names (nil or empty uses built-in allowlist).
func NewHtml2Text(tags []string) *Html2Text {
	tagMapper := make(map[string]struct{})
	if len(tags) == 0 {
		tags = defaultHtmlTags
	}
	for _, tag := range tags {
		tagMapper[tag] = struct{}{}
	}
	return &Html2Text{htmlTagMapper: tagMapper}
}

func (h *Html2Text) mapHtmlProperty(style string) map[string]string {
	mapper := make(map[string]string)
	style = strings.ToLower(style)
	styles := strings.Split(style, ";")
	for _, s := range styles {
		d := strings.SplitN(s, ":", 2)
		if len(d) == 2 {
			mapper[strings.Trim(d[0], " ")] = strings.Trim(d[1], " ")
		}
	}
	return mapper
}

// Parse returns text snippets and URLs found in HTML. Panics from parsing are recovered into nil slices.
func (h *Html2Text) Parse(src string) (text, url []string) {
	defer func() {
		if recover() != nil {
			text, url = nil, nil
		}
	}()
	if h == nil || h.htmlTagMapper == nil {
		return nil, nil
	}

	tagReplace := map[string]string{
		"<h1>":  "<span>",
		"<h2>":  "<span>",
		"<h3>":  "<span>",
		"<h4>":  "<span>",
		"<h5>":  "<span>",
		"<h6>":  "<span>",
		"<h7>":  "<span>",
		"</h1>": "</span>",
		"</h2>": "</span>",
		"</h3>": "</span>",
		"</h4>": "</span>",
		"</h5>": "</span>",
		"</h6>": "</span>",
		"</h7>": "</span>",
		"<H1>":  "<span>",
		"<H2>":  "<span>",
		"<H3>":  "<span>",
		"<H4>":  "<span>",
		"<H5>":  "<span>",
		"<H6>":  "<span>",
		"<H7>":  "<span>",
		"</H1>": "</span>",
		"</H2>": "</span>",
		"</H3>": "</span>",
		"</H4>": "</span>",
		"</H5>": "</span>",
		"</H6>": "</span>",
		"</H7>": "</span>",
	}
	for oldTag, newTag := range tagReplace {
		src = strings.ReplaceAll(src, oldTag, newTag)
	}
	document, err := goquery.NewDocumentFromReader(strings.NewReader(src))
	if err != nil {
		return nil, nil
	}
	document.Find("noscript").Remove()
	document.Find("script").Remove()
	document.Find("style").Remove()
	document.Find(`span[style*="display:none"]`).Remove()
	document.Find(`div[style*="display:none"]`).Remove()

	title := document.Find("title").Contents().Text()
	text = append(text, title)

	document.Find("body *").Each(func(i int, s *goquery.Selection) {
		node := s.Get(0)
		if node == nil {
			return
		}
		name := node.Data
		if _, ok := h.htmlTagMapper[name]; !ok {
			s.Empty()
		}

		if style, ok := s.Attr("style"); ok {
			m := h.mapHtmlProperty(style)
			width := m["width"]
			height := m["height"]
			font := m["font"]
			fontSize := m["font-size"]
			if strings.HasPrefix(width, "0") || strings.HasPrefix(width, "-") {
				s.Remove()
			}
			if strings.HasPrefix(height, "0") || strings.HasPrefix(height, "-") {
				s.Remove()
			}
			if strings.HasPrefix(font, "0") || strings.HasPrefix(font, "-") {
				s.Remove()
			}
			if strings.HasPrefix(fontSize, "0") || strings.HasPrefix(fontSize, "-") {
				s.Remove()
			}
			if m["display"] == "none" {
				s.Remove()
			}
		}
	})

	document.Find("body *").Each(func(i int, s *goquery.Selection) {
		if link, ok := s.Attr("href"); ok {
			linkText := s.Text()
			text = append(text, linkText)
			url = append(url, link)
		}
		if imgSrc, ok := s.Attr("src"); ok {
			imgAlt, _ := s.Attr("alt")
			if strings.Index(strings.ToLower(imgSrc), "data:") > 0 {
				url = append(url, imgSrc)
			}
			if len(imgAlt) > 0 {
				text = append(text, imgAlt)
			}
		}
	})

	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		if meta, ok := s.Attr("http-equiv"); ok {
			if strings.ToLower(meta) == "refresh" {
				if c, ok := s.Attr("text"); ok {
					d := strings.SplitN(c, "=", 2)
					if len(d) == 2 {
						url = append(url, d[1])
					}
				}
			}
		}
	})

	document.Find("video").Each(func(i int, s *goquery.Selection) {
		if poster, ok := s.Attr("poster"); ok {
			poster = strings.ReplaceAll(poster, "\r", "")
			poster = strings.ReplaceAll(poster, "\n", "")
			url = append(url, poster)
		}
	})

	document.Find("body").Each(func(i int, s *goquery.Selection) {
		innerText := s.Text()
		if len(innerText) > 0 {
			text = append(text, innerText)
		}
	})

	document.Find("input").Each(func(i int, s *goquery.Selection) {
		innerText, _ := s.Attr("value")
		if len(innerText) > 0 {
			text = append(text, innerText)
		}
	})

	return
}
