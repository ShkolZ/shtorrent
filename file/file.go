// package file

// import (
// 	"os"

// 	"github.com/ShkolZ/shtorrent/config"
// )

// type FileWriter struct {
// 	fileMap map[int][]*os.File
// }

// func InitializeFiles(cfg *config.Config) []*os.File {

// }

// func WriteToFile
package file

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/ShkolZ/shtorrent/config"
	"github.com/ShkolZ/shtorrent/piece"
)

type FileEntry struct {
	LowerOffset int
	UpperOffset int
	File        *os.File
}

func isSingleMode(cfg *config.Config) bool {
	if len(cfg.Torrent.Files) > 1 {
		return false
	} else {
		return true
	}
}

func InitializeFiles(cfg *config.Config) (chan *piece.Piece, error) {

	pieceDataCh := make(chan *piece.Piece, 10)

	if isSingleMode(cfg) {
		file, err := os.Create(cfg.Torrent.Name)
		if err != nil {
			return nil, fmt.Errorf("Some problem with creating file(SingleMode): %v\n", err)
		}
		file.Truncate(int64(cfg.Torrent.Length))
		go func() {
			for piece := range pieceDataCh {
				off := piece.Index * cfg.Torrent.PieceLength
				writeToFile(file, piece, int64(off))
			}
		}()

	} else {

		folderName := cfg.Torrent.Name
		os.Mkdir(folderName, 0755)
		currentOffset := 0
		fileArr := make([]FileEntry, 0)

		for _, f := range cfg.Torrent.Files {
			if strings.Contains(f.Path[0], "padding_file") {
				currentOffset += f.Length
			} else {
				path := path.Join(folderName, f.Path[0])
				file, err := os.Create(path)
				if err != nil {
					return nil, fmt.Errorf("Some error with creating files: %v\n", err)
				}
				file.Truncate(int64(f.Length))
				currFile := FileEntry{
					LowerOffset: currentOffset,
					UpperOffset: currentOffset + f.Length,
					File:        file,
				}
				fileArr = append(fileArr, currFile)
				currentOffset += f.Length
			}
		}

		go func() {
			for piece := range pieceDataCh {
				for _, file := range fileArr {
					pieceOffset := piece.Index * cfg.Torrent.PieceLength
					if pieceOffset >= file.LowerOffset && pieceOffset < file.UpperOffset {
						if pieceOffset+cfg.Torrent.Length > file.UpperOffset {
							limit := file.UpperOffset - pieceOffset
							piece.Data = piece.Data[:limit]
						}
						realOff := pieceOffset - file.LowerOffset
						writeToFile(file.File, piece, int64(realOff))
					}
				}
			}
		}()
		fmt.Println(fileArr)
	}
	return pieceDataCh, nil
}

func calculateOffsets() {

}

func writeToFile(file *os.File, p *piece.Piece, off int64) {
	length := len(p.Data)
	written := 0
	for written < length {
		n, err := file.WriteAt(p.Data[written:], off+int64(written))
		if err != nil && err != io.EOF {
			fmt.Println("Some problem with writing file")
		}
		written += n

	}

	fmt.Printf("Wrote Piece %d at offset: %v with length: %v\n", p.Index, off/1024, length/1024)

}
