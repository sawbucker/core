package web

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/ShoshinNikita/log"
	"github.com/pkg/errors"

	"github.com/tags-drive/core/internal/params"
	"github.com/tags-drive/core/internal/storage/files"
)

const (
	maxSize = 50000000 // 50MB
)

var (
	ErrEmptyFilename = errors.New("name of a file can't be empty")
)

// multiplyResponse is used as response by POST /api/files and DELETE /api/files
type multiplyResponse struct {
	Filename string `json:"filename"`
	IsError  bool   `json:"isError"`
	Error    string `json:"error"`
	Status   string `json:"status"` // Status isn't empty when IsError == false
}

// GET /api/files?sort=(name|size|time)&order(asc|desc)&tags=first,second,third&mode=(or|and|not)&search=abc
// tags - list of tags separated by ',' (can be empty, then all files will be returned)
// First elements in params is default (name, asc and etc.)
//
// Response: json array of files
//
func returnFiles(w http.ResponseWriter, r *http.Request) {
	var (
		order = getParam("asc", r.FormValue("order"), "asc", "desc")
		tags  = func() []int {
			t := r.FormValue("tags")
			if t == "" {
				return []int{}
			}
			res := []int{}

			for _, s := range strings.Split(t, ",") {
				if id, err := strconv.Atoi(s); err == nil {
					res = append(res, id)
				}
			}
			return res
		}()
		search = r.FormValue("search")

		tagMode  = files.ModeOr
		sortMode = files.SortByNameAsc
	)

	// Set sortMode
	// Can skip default
	switch r.FormValue("sort") {
	case "name":
		if order == "asc" {
			sortMode = files.SortByNameAsc
		} else {
			sortMode = files.SortByNameDesc
		}
	case "size":
		if order == "asc" {
			sortMode = files.SortBySizeAsc
		} else {
			sortMode = files.SortBySizeDecs
		}
	case "time":
		if order == "asc" {
			sortMode = files.SortByTimeAsc
		} else {
			sortMode = files.SortByTimeDesc
		}
	}

	// Set tagMode
	// Can skip default
	switch r.FormValue("mode") {
	case "or":
		tagMode = files.ModeOr
	case "and":
		tagMode = files.ModeAnd
	case "not":
		tagMode = files.ModeNot
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}

	enc.Encode(files.Get(tagMode, sortMode, tags, search))
}

func getParam(def, passed string, options ...string) (s string) {
	s = def
	if passed == def {
		return
	}
	for _, opt := range options {
		if passed == opt {
			return passed
		}
	}

	return
}

// GET /api/files/download?file=132.png&file=456.jpg
//
// Response: blob zip-file
//
func downloadFiles(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filenames := r.Form["file"]
	if len(filenames) == 0 {
		Error(w, "list of files can't be empty", http.StatusBadRequest)
		return
	}

	body, err := files.ArchiveFiles(filenames)
	if err != nil {
		Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	if _, err := io.Copy(w, body); err != nil {
		log.Errorf("can't copy zip file to response body: %s\n", err)
	}
}

// GET /api/files/recent?number=5 (5 is a default value)
//
// Response: json array of files
//
func returnRecentFiles(w http.ResponseWriter, r *http.Request) {
	number := func() int {
		s := r.FormValue("number")
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			n = 5
		}
		return int(n)
	}()

	files := files.GetRecent(number)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(files)
}

// POST /api/files (multipart/form-data)
//
// Response: json list of strings with status of files uploading
//
func upload(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		switch err {
		case http.ErrNotMultipart:
			Error(w, err.Error(), http.StatusBadRequest)
		default:
			Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	respChan := make(chan multiplyResponse, 50)
	wg := new(sync.WaitGroup)

	for _, f := range r.MultipartForm.File["files"] {
		wg.Add(1)
		go func(header *multipart.FileHeader) {
			defer wg.Done()
			err := files.UploadFile(header)
			if err != nil {
				respChan <- multiplyResponse{Filename: header.Filename, IsError: true, Error: err.Error()}
			} else {
				respChan <- multiplyResponse{Filename: header.Filename, Status: "uploaded"}
			}
		}(f)
	}
	wg.Wait()
	close(respChan)

	var response []multiplyResponse
	for resp := range respChan {
		// Log an error
		if resp.IsError {
			log.Errorf("Can't load a file %s: %s\n", resp.Filename, resp.Error)
		}
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(response)
}

// PUT /api/files/name?file=123&new-name=567
//
// Response: -
//
func changeFilename(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")
	if filename == "" {
		Error(w, ErrEmptyFilename.Error(), http.StatusBadRequest)
		return
	}

	newName := r.FormValue("new-name")
	if newName == "" {
		Error(w, "new-name param can't be empty", http.StatusBadRequest)
		return
	}

	// We can skip checking of invalid characters, because Go will return an error
	err := files.RenameFile(filename, newName)
	if err != nil {
		Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// PUT /api/files/tags?file=123&tags=1,2,3
//
// Response: -
//
func changeFileTags(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")
	if filename == "" {
		Error(w, ErrEmptyFilename.Error(), http.StatusBadRequest)
		return
	}

	tags := func() []int {
		t := r.FormValue("tags")
		if t == "" {
			return []int{}
		}

		res := []int{}
		for _, s := range strings.Split(t, ",") {
			if id, err := strconv.Atoi(s); err == nil {
				res = append(res, id)
			}
		}
		return res
	}()

	err := files.ChangeTags(filename, tags)
	if err != nil {
		Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// PUT /api/files/description?file=123&description=some-new-cool-description
//
// Response: -
//
func changeFileDescription(w http.ResponseWriter, r *http.Request) {
	filename := r.FormValue("file")
	if filename == "" {
		Error(w, ErrEmptyFilename.Error(), http.StatusBadRequest)
		return
	}

	newDescription := r.FormValue("description")
	err := files.ChangeDescription(filename, newDescription)
	if err != nil {
		Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DELETE /api/files?file=file1&file=file2
//
// Response: -
//
func deleteFile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	filenames := r.Form["file"]
	if len(filenames) == 0 {
		Error(w, "list of files for deleting can't be empty", http.StatusBadRequest)
		return
	}

	respChan := make(chan multiplyResponse, 50)
	wg := new(sync.WaitGroup)

	for _, filename := range filenames {
		wg.Add(1)
		go func(f string) {
			defer wg.Done()
			err := files.DeleteFile(f)
			if err != nil {
				respChan <- multiplyResponse{Filename: f, IsError: true, Error: err.Error()}
			} else {
				respChan <- multiplyResponse{Filename: f, Status: "deleted"}
			}
		}(filename)
	}

	wg.Wait()
	close(respChan)

	var response []multiplyResponse
	for resp := range respChan {
		response = append(response, resp)
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(response)
}
