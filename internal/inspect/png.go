package inspect

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"strings"
)

const (
	maximumCompressedTextBytes = 1024 * 1024
	maximumMetadataChunkBytes  = 8 * 1024 * 1024
	maximumPNGChunks           = 10000
)

func parsePNG(data []byte, collector *collector) (int, int, error) {
	if len(data) < len(pngSignature) || string(data[:len(pngSignature)]) != string(pngSignature) {
		return 0, 0, fmt.Errorf("malformed PNG: missing signature")
	}

	width, height := 0, 0
	foundEnd := false
	chunkCount := 0
	for position := len(pngSignature); position < len(data); {
		chunkCount++
		if chunkCount > maximumPNGChunks {
			return 0, 0, fmt.Errorf("malformed PNG: exceeds the %d-chunk safety limit", maximumPNGChunks)
		}
		if position+12 > len(data) {
			return 0, 0, fmt.Errorf("malformed PNG: incomplete chunk at byte %d", position)
		}
		chunkLength := uint64(binary.BigEndian.Uint32(data[position : position+4]))
		chunkEnd := uint64(position) + 12 + chunkLength
		if chunkEnd > uint64(len(data)) {
			return 0, 0, fmt.Errorf("malformed PNG: chunk exceeds file size")
		}

		chunkTypeBytes := data[position+4 : position+8]
		chunkType := string(chunkTypeBytes)
		chunkData := data[position+8 : position+8+int(chunkLength)]
		expectedCRC := binary.BigEndian.Uint32(data[position+8+int(chunkLength) : int(chunkEnd)])
		checksum := crc32.NewIEEE()
		_, _ = checksum.Write(chunkTypeBytes)
		_, _ = checksum.Write(chunkData)
		if checksum.Sum32() != expectedCRC {
			collector.warn("PNG chunk %s has an invalid CRC.", safeText(chunkType))
		}

		if isPNGMetadataChunk(chunkType) && len(chunkData) > maximumMetadataChunkBytes {
			collector.warn(
				"Ignored PNG %s metadata larger than the %d MiB safety limit.",
				safeText(chunkType), maximumMetadataChunkBytes/(1024*1024),
			)
			position = int(chunkEnd)
			continue
		}

		switch chunkType {
		case "IHDR":
			if len(chunkData) != 13 {
				return 0, 0, fmt.Errorf("malformed PNG: invalid IHDR length")
			}
			width = int(binary.BigEndian.Uint32(chunkData[0:4]))
			height = int(binary.BigEndian.Uint32(chunkData[4:8]))
		case "eXIf":
			collector.addContainer("PNG eXIf")
			if err := parseEXIF(chunkData, "PNG eXIf", collector); err != nil {
				collector.warn("Could not parse PNG EXIF metadata: %v", err)
			}
		case "tEXt":
			collector.addContainer("PNG tEXt")
			keyword, value, err := parseTextChunk(chunkData)
			if err != nil {
				collector.warn("Could not parse a PNG tEXt chunk: %v", err)
			} else {
				parsePNGText(keyword, value, "tEXt", collector)
			}
		case "zTXt":
			collector.addContainer("PNG zTXt")
			keyword, value, err := parseCompressedTextChunk(chunkData)
			if err != nil {
				collector.warn("Could not parse a PNG zTXt chunk: %v", err)
			} else {
				parsePNGText(keyword, value, "zTXt", collector)
			}
		case "iTXt":
			collector.addContainer("PNG iTXt")
			keyword, value, err := parseInternationalTextChunk(chunkData)
			if err != nil {
				collector.warn("Could not parse a PNG iTXt chunk: %v", err)
			} else {
				parsePNGText(keyword, value, "iTXt", collector)
			}
		case "IEND":
			foundEnd = true
		}

		position = int(chunkEnd)
		if foundEnd {
			break
		}
	}

	if width == 0 || height == 0 {
		return 0, 0, fmt.Errorf("malformed PNG: missing or invalid IHDR")
	}
	if !foundEnd {
		return 0, 0, fmt.Errorf("malformed PNG: missing IEND")
	}
	return width, height, nil
}

func isPNGMetadataChunk(chunkType string) bool {
	return chunkType == "eXIf" || chunkType == "tEXt" || chunkType == "zTXt" || chunkType == "iTXt"
}

func parseTextChunk(data []byte) (string, string, error) {
	separator := bytes.IndexByte(data, 0)
	if separator <= 0 {
		return "", "", fmt.Errorf("missing keyword separator")
	}
	return string(data[:separator]), string(data[separator+1:]), nil
}

func parseCompressedTextChunk(data []byte) (string, string, error) {
	separator := bytes.IndexByte(data, 0)
	if separator <= 0 || separator+2 > len(data) {
		return "", "", fmt.Errorf("invalid compressed text header")
	}
	if data[separator+1] != 0 {
		return "", "", fmt.Errorf("unsupported compression method %d", data[separator+1])
	}
	value, err := decompressText(data[separator+2:])
	return string(data[:separator]), value, err
}

func parseInternationalTextChunk(data []byte) (string, string, error) {
	keywordEnd := bytes.IndexByte(data, 0)
	if keywordEnd <= 0 || keywordEnd+3 > len(data) {
		return "", "", fmt.Errorf("invalid international text header")
	}
	keyword := string(data[:keywordEnd])
	compressionFlag := data[keywordEnd+1]
	compressionMethod := data[keywordEnd+2]
	remainder := data[keywordEnd+3:]

	languageEnd := bytes.IndexByte(remainder, 0)
	if languageEnd < 0 {
		return "", "", fmt.Errorf("missing language separator")
	}
	remainder = remainder[languageEnd+1:]
	translatedEnd := bytes.IndexByte(remainder, 0)
	if translatedEnd < 0 {
		return "", "", fmt.Errorf("missing translated-keyword separator")
	}
	remainder = remainder[translatedEnd+1:]

	switch compressionFlag {
	case 0:
		return keyword, string(remainder), nil
	case 1:
		if compressionMethod != 0 {
			return "", "", fmt.Errorf("unsupported compression method %d", compressionMethod)
		}
		value, err := decompressText(remainder)
		return keyword, value, err
	default:
		return "", "", fmt.Errorf("invalid compression flag %d", compressionFlag)
	}
}

func decompressText(data []byte) (string, error) {
	reader, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer reader.Close()

	decoded, err := io.ReadAll(io.LimitReader(reader, maximumCompressedTextBytes+1))
	if err != nil {
		return "", err
	}
	if len(decoded) > maximumCompressedTextBytes {
		return "", fmt.Errorf("decompressed text exceeds %d bytes", maximumCompressedTextBytes)
	}
	return string(decoded), nil
}

func parsePNGText(keyword, value, chunkType string, collector *collector) {
	keyword = safeText(keyword)
	source := fmt.Sprintf("PNG %s:%s", chunkType, keyword)
	switch strings.ToLower(keyword) {
	case "author", "artist":
		collector.addValue(&collector.metadata.Authors, value, source)
	case "copyright":
		collector.addValue(&collector.metadata.Copyright, value, source)
	case "software":
		collector.addValue(&collector.metadata.Software, value, source)
	case "comment", "description", "caption":
		collector.addValue(&collector.metadata.Comments, value, source)
	case "xml:com.adobe.xmp":
		collector.addContainer("PNG XMP")
		if err := parseXMP([]byte(value), "PNG XMP", collector); err != nil {
			collector.warn("Could not parse PNG XMP metadata: %v", err)
		}
	}
}
