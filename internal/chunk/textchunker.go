package chunk

import (
	"context"
	"log"
	"strings"
)

type TextChunker struct {
	Maxsize      int
	Minsize      int
	ChunkOverlap float64
}

func NewTextChunker(maxsize int, minsize int, overlap float64) Chunker {
	return &TextChunker{
		Maxsize:      maxsize,
		Minsize:      minsize,
		ChunkOverlap: overlap,
	}
}

func (tc *TextChunker) Chunk(ctx context.Context, link string, content string) ([]Chunk, error) {
	if strings.TrimSpace(content) == "" {
		return []Chunk{}, nil
	}

	var chunks []Chunk
	paragraphs := tc.splitIntoParagraphs(content)

	currentChunk := ""
	chunkIndex := 0

	for _, paragraph := range paragraphs {
		// Try adding the whole paragraph
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += "\n\n" + paragraph
		} else {
			testChunk = paragraph
		}

		testTokenCount := simpleTokenCount(testChunk)

		if testTokenCount <= tc.Maxsize {
			// Paragraph fits, add it
			currentChunk = testChunk
		} else if currentChunk != "" {
			// Current chunk is full, save it and start new one
			tokenCount := simpleTokenCount(currentChunk)
			if tokenCount >= tc.Minsize {
				chunks = append(chunks, Chunk{
					Content:    strings.TrimSpace(currentChunk),
					Link:       link,
					TokenCount: tokenCount,
				})
				chunkIndex++
			}

			// Start new chunk with overlap
			overlapText := tc.getOverlapText(currentChunk)
			if overlapText != "" && simpleTokenCount(overlapText+"\n\n"+paragraph) <= tc.Maxsize {
				currentChunk = overlapText + "\n\n" + paragraph
			} else {
				currentChunk = paragraph
			}

			// If single paragraph is still too large, split by sentences
			if simpleTokenCount(currentChunk) > tc.Maxsize {
				sentenceChunks := tc.chunkBySentences(paragraph, link, &chunkIndex)
				chunks = append(chunks, sentenceChunks...)
				currentChunk = ""
			}
		} else {
			// First paragraph is too large, split by sentences
			sentenceChunks := tc.chunkBySentences(paragraph, link, &chunkIndex)
			chunks = append(chunks, sentenceChunks...)
		}
	}

	// Add final chunk if it exists and meets minimum size
	if currentChunk != "" {
		tokenCount := simpleTokenCount(currentChunk)
		if tokenCount >= tc.Minsize {
			chunks = append(chunks, Chunk{
				Content:    strings.TrimSpace(currentChunk),
				TokenCount: tokenCount,
				Link:       link,
			})
		}
	}

	log.Printf("chunk succeeded with %v results", len(chunks))
	return chunks, nil
}

// chunkBySentences handles chunking when paragraphs are too large
func (tc *TextChunker) chunkBySentences(text string, link string, chunkIndex *int) []Chunk {
	var chunks []Chunk
	sentences := tc.splitIntoSentences(text)

	currentChunk := ""

	for _, sentence := range sentences {
		testChunk := currentChunk
		if testChunk != "" {
			testChunk += " " + sentence
		} else {
			testChunk = sentence
		}

		testTokenCount := simpleTokenCount(testChunk)

		if testTokenCount <= tc.Maxsize {
			currentChunk = testChunk
		} else if currentChunk != "" {
			// Save current chunk
			tokenCount := simpleTokenCount(currentChunk)
			if tokenCount >= tc.Minsize {
				chunks = append(chunks, Chunk{
					Content:    strings.TrimSpace(currentChunk),
					TokenCount: tokenCount,
					Link:       link,
				})
				*chunkIndex++
			}

			// Start new chunk with overlap
			overlapText := tc.getOverlapText(currentChunk)
			if overlapText != "" && simpleTokenCount(overlapText+" "+sentence) <= tc.Maxsize {
				currentChunk = overlapText + " " + sentence
			} else {
				currentChunk = sentence
			}
		} else {
			// Single sentence is too large, keep it anyway
			currentChunk = sentence
		}
	}

	// Add final sentence chunk
	if currentChunk != "" {
		tokenCount := simpleTokenCount(currentChunk)
		if tokenCount >= tc.Minsize {
			chunks = append(chunks, Chunk{
				Content:    strings.TrimSpace(currentChunk),
				TokenCount: tokenCount,
				Link:       link,
			})
			*chunkIndex++
		}
	}

	return chunks
}

func (tc *TextChunker) splitIntoParagraphs(text string) []string {
	paragraphs := strings.Split(text, "\n")

	// Filter empty paragraphs
	var filteredParagraphs []string
	for _, p := range paragraphs {
		p = strings.TrimSpace(p)
		if p != "" {
			filteredParagraphs = append(filteredParagraphs, p)
		}
	}
	return filteredParagraphs
}

func (tc *TextChunker) splitIntoSentences(text string) []string {
	var sentences []string
	var currentSentence string

	for _, r := range text {
		currentSentence += string(r)
		// End of sentence
		if r == '.' || r == '!' || r == '?' {
			sentences = append(sentences, strings.TrimSpace(currentSentence))
			currentSentence = ""
		}
	}

	if currentSentence != "" {
		sentences = append(sentences, strings.TrimSpace(currentSentence))
	}
	return sentences
}

// getOverlapText extracts overlap text from the end of a chunk
func (tc *TextChunker) getOverlapText(text string) string {
	overlapTokens := int(float64(tc.Maxsize) * tc.ChunkOverlap)

	// Try to get overlap by sentences first
	sentences := tc.splitIntoSentences(text)
	if len(sentences) == 0 {
		return ""
	}

	overlapText := ""

	// Start from the end and work backwards
	for i := len(sentences) - 1; i >= 0; i-- {
		testText := sentences[i]
		if overlapText != "" {
			testText = sentences[i] + " " + overlapText
		}

		if simpleTokenCount(testText) <= overlapTokens {
			overlapText = testText
		} else {
			break
		}
	}

	// If no good sentence overlap, use character-based overlap
	if overlapText == "" {
		overlapChars := overlapTokens * 4
		if len(text) > overlapChars {
			overlapText = text[len(text)-overlapChars:]
			// Try to start at a word boundary
			if spaceIdx := strings.Index(overlapText, " "); spaceIdx != -1 {
				overlapText = overlapText[spaceIdx+1:]
			}
		} else {
			overlapText = text
		}
	}

	return overlapText
}

func simpleTokenCount(content string) int {
	return (len(content) / 4)
}
