package main

import (
  "image"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/tiff"
  "github.com/sergeymakinen/go-bmp"
)

type ImageData struct {
	FileName string
	Properties map[string]string
}

func main() {
  http.HandleFunc("/", serveHTML)
  http.HandleFunc("/upload", handleUpload)

	log.Println("Server started at :8081")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func serveHTML(w http.ResponseWriter, r *http.Request) {
    folderPath := "./uploads" // Change this to your folder path
		exifDataList := readExifData(folderPath)
		tmpl := template.Must(template.New("exifTable").Parse(exifTableTemplate))
		tmpl.Execute(w, exifDataList)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
    err := r.ParseMultipartForm(10 << 20) // 10 MB max memory
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    files := r.MultipartForm.File["folder"]
    if len(files) == 0 {
        http.Error(w, "No files uploaded", http.StatusBadRequest)
        return
    }

    // Extract the folder name from the first file header
    folderName := extractFolderName(files[0].Filename)
    fmt.Printf("Folder Name: %s\n", folderName)

    // Create a directory to store the uploaded files
    uploadDir := filepath.Join("uploads", folderName)
    err = os.MkdirAll(uploadDir, os.ModePerm)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    for _, fileHeader := range files {
        file, err := fileHeader.Open()
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer file.Close()

        out, err := os.Create(filepath.Join(uploadDir, fileHeader.Filename))
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        defer out.Close()

        _, err = io.Copy(out, file)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
    }

    folderPath := uploadDir // Change this to your folder path
		exifDataList := readExifData(folderPath)
		tmpl := template.Must(template.New("exifTable").Parse(exifTableTemplate))
		tmpl.Execute(w, exifDataList)
}

func extractFolderName(filePath string) string {
    // Split the file path and return the folder name
    parts := strings.Split(filePath, string(os.PathSeparator))
    if len(parts) > 1 {
        return parts[0]
    }
    return "unknown_folder"
}

func readExifData(folderPath string) []ImageData {
	var imageDataList []ImageData

	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Printf("Error accessing path %s: %v\n", path, err)
			return nil
		}
		if !info.IsDir() {
			imageData, err := getImageData(path)
			if err != nil {
				fmt.Printf("Error reading EXIF data from %s: %v\n", path, err)
				return nil
			}
      fmt.Println(len(imageData.Properties))
			imageDataList = append(imageDataList, imageData)
      fmt.Println(len(imageDataList))
		}
		return nil
	})

	return imageDataList
}


type MyWalker struct {
  mp map[string]string
}

func (w MyWalker) Walk (name exif.FieldName, tag *tiff.Tag) error {
  var err error
  w.mp[string(name)], err = tag.StringVal()
  if err != nil {
    return nil
  }

  return nil
}

func getImageData(filePath string) (ImageData, error) {
  fmt.Println("reading data from ", filePath)
	file, err := os.Open(filePath)
	if err != nil {
    fmt.Println("can't open file ", filePath)
		return ImageData{}, err
	}
	defer file.Close()

  ext := strings.ToLower(filepath.Ext(filePath));
  var walker MyWalker = MyWalker{make(map[string]string)}
  switch ext {
  case ".bmp":
    var image image.Image
    image, err := bmp.Decode(file);
  	if err != nil {
		  return ImageData{}, err
	  }
    walker.mp["resolution x"] = fmt.Sprint(image.Bounds().Size().X)
    walker.mp["resolution y"] = fmt.Sprint(image.Bounds().Size().Y)
  default:
    exifData, err := exif.Decode(file)
    exifData.Walk(walker)
  	if err != nil {
		  return ImageData{}, err
	  }
  }
  fmt.Println("writing data from ", filePath)
	return ImageData{FileName: filePath, Properties: walker.mp}, nil
}

const exifTableTemplate = `
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Image Data</title>
    <style>
        table {
            width: 100%;
            border-collapse: collapse;
        }
        th, td {
            border: 1px solid black;
            padding: 8px;
            text-align: left;
        }
        th {
            background-color: #f2f2f2;
        }
    </style>
</head>
<body>
    <h1>Select a Folder</h1>
        <form id="folderForm" action="/upload" method="post" enctype="multipart/form-data">
        <input type="file" id="folderInput" name="folder" webkitdirectory directory multiple>
        <button type="submit">Upload</button>
    </form>
    <h1>Data</h1>
    <table>
        <tr>
            <th>File Name</th>
            <th>Data</th>
        </tr>
        {{range .}}
        <tr>
            <td>{{.FileName}}</td>
            <td>
                <table>
                    {{range $key, $value := .Properties}}
                    <tr>
                        <td>{{$key}}</td>
                        <td>{{$value}}</td>
                    </tr>
                    {{end}}
                </table>
            </td>
        </tr>
        {{end}}
    </table>
</body>
</html>
`
