package web

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	"github.com/tags-drive/core/cmd"
	"github.com/tags-drive/core/internal/params"
	filesPck "github.com/tags-drive/core/internal/storage/files"
	"github.com/tags-drive/core/internal/storage/files/aggregation"
)

const (
	// It is trade-off between memory and I/O
	// If maxSize == 50MB, program takes too much memory
	// If maxSize == 2MB, there're too many I/O-operations
	maxSize = 10 << 20 // 10MB
)

// multiplyResponse is used as a response by POST /api/files and DELETE /api/files
type multiplyResponse struct {
	Filename string `json:"filename"`
	IsError  bool   `json:"isError"`
	Error    string `json:"error"`
	Status   string `json:"status"` // Status isn't empty when IsError == false
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

// GET /api/file/{id}
//
// Params:
//   - id: id of a file
//
// Response: json object
//
func (s Server) returnSingleFile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.processError(w, "invalid id", http.StatusBadRequest)
		return
	}

	file, err := s.fileStorage.GetFile(id)
	if err != nil {
		if err == filesPck.ErrFileIsNotExist {
			s.processError(w, "file doesn't exist", http.StatusNotFound)
			return
		}

		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}

	enc.Encode(file)
}

// GET /api/files
//
// Params:
//   - expr: logical expression
//   - search: text for search
//   - regexp: is search a regular expression (it is true when regexp != "")
//   - sort: name | size | time
//   - order: asc | desc
//   - offset: lower bound [offset:]
//   - count: number of returned files ([offset:offset+count]). If count == 0, all files will be returned. Default is 0
//
// Response: json array
//
func (s Server) returnFiles(w http.ResponseWriter, r *http.Request) {
	var (
		order    = getParam("asc", r.FormValue("order"), "asc", "desc")
		expr     = r.FormValue("expr")
		search   = r.FormValue("search")
		isRegexp = r.FormValue("regexp") != ""
		offset   = 0
		count    = 0
		sortMode = cmd.SortByNameAsc
	)

	// Check is regexp valid
	if isRegexp {
		_, err := regexp.Compile(search)
		if err != nil {
			s.processError(w, "invalid regular expression", http.StatusBadRequest)
			return
		}
	}

	// Get offset
	offset = func() int {
		param := r.FormValue("offset")
		if param == "" {
			return 0
		}

		r, err := strconv.Atoi(param)
		if err != nil || r < 0 {
			return 0
		}

		return r
	}()

	// Get offset
	count = func() int {
		param := r.FormValue("count")
		if param == "" {
			return 0
		}

		r, err := strconv.Atoi(param)
		if err != nil || r < 0 {
			return 0
		}

		return r
	}()

	// Set sortMode
	// Can skip default
	switch r.FormValue("sort") {
	case "name":
		if order == "asc" {
			sortMode = cmd.SortByNameAsc
		} else {
			sortMode = cmd.SortByNameDesc
		}
	case "size":
		if order == "asc" {
			sortMode = cmd.SortBySizeAsc
		} else {
			sortMode = cmd.SortBySizeDecs
		}
	case "time":
		if order == "asc" {
			sortMode = cmd.SortByTimeAsc
		} else {
			sortMode = cmd.SortByTimeDesc
		}
	}

	files, err := s.fileStorage.Get(expr, sortMode, search, isRegexp, offset, count)
	if err != nil {
		if err == filesPck.ErrOffsetOutOfBounds {
			w.WriteHeader(http.StatusNoContent)
			fmt.Fprint(w, err.Error())
		} else if err == aggregation.ErrBadSyntax {
			s.processError(w, err.Error(), http.StatusBadRequest)
		} else {
			s.processError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}

	enc.Encode(files)
}

// GET /api/files/recent
//
// Params:
//   - number: number of returned files (5 is a default value)
//
// Response: same as `GET /api/files`
//
func (s Server) returnRecentFiles(w http.ResponseWriter, r *http.Request) {
	number := func() int {
		s := r.FormValue("number")
		n, err := strconv.ParseUint(s, 10, 32)
		if err != nil {
			n = 5
		}
		return int(n)
	}()

	files := s.fileStorage.GetRecent(number)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(files)
}

// GET /api/files/download
//
// Params:
//   - ids: list of ids of files for downloading separated by comma `ids=1,2,54,9`
//
// Response: zip archive
//
func (s Server) downloadFiles(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ids := func() (res []int) {
		strIDs := r.FormValue("ids")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			if err == nil {
				res = append(res, id)
			}
		}
		return
	}()

	body, err := s.fileStorage.Archive(ids)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/zip")
	if _, err := io.Copy(w, body); err != nil {
		s.logger.Errorf("can't copy zip file to response body: %s\n", err)
	}
}

// POST /api/files
//
// Body must be "multipart/form-data"
//
// Params:
//   - tags: list of tags, separated by comma (`tags=1,2,3`)
//
// Response: json array
//
func (s Server) upload(w http.ResponseWriter, r *http.Request) {
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

	err := r.ParseMultipartForm(maxSize)
	if err != nil {
		switch err {
		case http.ErrNotMultipart:
			s.processError(w, err.Error(), http.StatusBadRequest)
		default:
			s.processError(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	headersChan := make(chan interface{}, 5)
	// Fill headersChan
	go func() {
		for i := range r.MultipartForm.File["files"] {
			headersChan <- r.MultipartForm.File["files"][i]
		}
		close(headersChan)
	}()

	runPool(5, headersChan, func(data <-chan interface{}) {
		for d := range data {
			header, ok := d.(*multipart.FileHeader)
			if !ok {
				continue
			}

			err := s.fileStorage.Upload(header, tags)
			var resp multiplyResponse
			if err != nil {
				resp = multiplyResponse{
					Filename: header.Filename,
					IsError:  true,
					Error:    err.Error(),
				}
				s.logger.Errorf("can't load a file %s: %s\n", header.Filename, err)
			} else {
				resp = multiplyResponse{Filename: header.Filename, Status: "uploaded"}
			}

			data, _ := json.Marshal(resp)
			s.sessionStorage.Broadcast(data)
		}
	})
}

// POST /api/files/recover
//
// Params:
//   - ids: list ids of files for recovering separated by comma `ids=1,2,54,9`
//
// Response: -
//
func (s Server) recoverFile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ids := func() (res []int) {
		strIDs := r.FormValue("ids")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			if err == nil {
				res = append(res, id)
			}
		}
		return
	}()

	if len(ids) == 0 {
		s.processError(w, "list of ids of files for recovering can't be empty", http.StatusBadRequest)
		return
	}

	idsChan := make(chan interface{}, 5)
	go func() {
		for i := range ids {
			idsChan <- ids[i]
		}
		close(idsChan)
	}()

	runPool(5, idsChan, func(data <-chan interface{}) {
		for d := range data {
			id, ok := d.(int)
			if !ok {
				continue
			}
			s.fileStorage.Recover(id)
		}
	})
}

// PUT /api/file/{id}/name
//
// Params:
//   - id: file id
//   - new-name: new filename
//
//  Response: updated file
//
func (s Server) changeFilename(w http.ResponseWriter, r *http.Request) {
	strID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(strID)
	if err != nil {
		s.processError(w, "bad id syntax", http.StatusBadRequest)
		return
	}

	newName := r.FormValue("new-name")
	if newName == "" {
		s.processError(w, "new-name param can't be empty", http.StatusBadRequest)
		return
	}

	// We can skip checking of invalid characters, because Go will return an error
	updatedFile, err := s.fileStorage.Rename(id, newName)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(updatedFile)
}

// PUT /api/file/{id}/tags
//
// Params:
//   - id: file id
//   - tags: updated list of tags, separated by comma (`tags=1,2,3`)
//
// Response: updated file
//
func (s Server) changeFileTags(w http.ResponseWriter, r *http.Request) {
	strID := mux.Vars(r)["id"]
	fileID, err := strconv.Atoi(strID)
	if err != nil {
		s.processError(w, "bad id syntax", http.StatusBadRequest)
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

	var goodTags []int
	for _, id := range tags {
		if s.tagStorage.Check(id) {
			goodTags = append(goodTags, id)
		}
	}

	updatedFile, err := s.fileStorage.ChangeTags(fileID, goodTags)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(updatedFile)
}

// PUT /api/file/{id}/description
//
// Params:
//   - id: file id
//   - description: updated description
//
// Response: updated file
//
func (s Server) changeFileDescription(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		s.processError(w, "bad id syntax", http.StatusBadRequest)
		return
	}
	newDescription := r.FormValue("description")

	updatedFile, err := s.fileStorage.ChangeDescription(id, newDescription)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	if params.Debug {
		enc.SetIndent("", "  ")
	}
	enc.Encode(updatedFile)
}

// POST /api/files/tags
//
// Params:
//   - files: file ids (list of ids separated by ',')
//   - tags: tags for adding (list of tags ids separated by ',')
//
// Response: -
//
func (s Server) addTagsToFiles(w http.ResponseWriter, r *http.Request) {
	filesIDs := func() (res []int) {
		strIDs := r.FormValue("files")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			if err == nil {
				res = append(res, id)
			}
		}
		return res
	}()

	tagsIDs := func() (res []int) {
		strIDs := r.FormValue("tags")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			// Add only valid tags
			if err == nil && s.tagStorage.Check(id) {
				res = append(res, int(id))
			}
		}
		return res
	}()

	s.fileStorage.AddTagsToFiles(filesIDs, tagsIDs)
}

// DELETE /api/files/tags
//
// Params:
//   - files: file ids (list of ids separated by ',')
//   - tags: tags for deleting (list of tags ids separated by ',')
//
// Response: -
//
func (s Server) removeTagsFromFiles(w http.ResponseWriter, r *http.Request) {
	filesIDs := func() (res []int) {
		strIDs := r.FormValue("files")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			if err == nil {
				res = append(res, id)
			}
		}
		return res
	}()

	tagsIDs := func() (res []int) {
		strIDs := r.FormValue("tags")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			// Add only valid tags
			if err == nil && s.tagStorage.Check(id) {
				res = append(res, int(id))
			}
		}
		return res
	}()

	s.fileStorage.RemoveTagsFromFiles(filesIDs, tagsIDs)
}

// DELETE /api/files
//
// Params:
//   - ids: list of ids of files for deleting separated by comma `ids=1,2,54,9`
//   - force: should file be deleted right now
//     (if it isn't empty, file will be deleted right now)
//
// Response: json array
//
func (s Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ids := func() (res []int) {
		strIDs := r.FormValue("ids")
		for _, strID := range strings.Split(strIDs, ",") {
			id, err := strconv.Atoi(strID)
			if err == nil {
				res = append(res, id)
			}
		}
		return
	}()

	force := func() bool {
		return r.FormValue("force") != ""
	}()

	filesIDsChan := make(chan interface{}, 5)
	// Fill filesIDsChan
	go func() {
		for _, id := range ids {
			filesIDsChan <- id
		}
		close(filesIDsChan)
	}()

	// Used in a worker function
	var (
		deleteFunc = s.fileStorage.Delete
		// We will use status if deleteFunc returns nil error
		respStatus = "added into trash"
	)

	if force {
		deleteFunc = s.fileStorage.DeleteForce
		respStatus = "deleted"
	}

	runPool(5, filesIDsChan, func(data <-chan interface{}) {
		for d := range data {
			id, ok := d.(int)
			if !ok {
				continue
			}

			var resp multiplyResponse

			// Check file
			file, err := s.fileStorage.GetFile(id)
			if err != nil {
				msg := err.Error()
				if err == filesPck.ErrFileIsNotExist {
					msg = fmt.Sprintf("file with id \"%d\" doesn't exist", id)
				}

				resp = multiplyResponse{
					Filename: "",
					IsError:  true,
					Error:    msg,
				}
			} else {
				// Delete file
				err = deleteFunc(id)
				if err != nil {
					resp = multiplyResponse{
						Filename: file.Filename,
						IsError:  true,
						Error:    err.Error(),
					}
				} else {
					// Use pre-defined var status
					resp = multiplyResponse{
						Filename: file.Filename,
						Status:   respStatus,
					}
				}
			}

			data, _ := json.Marshal(resp)
			s.sessionStorage.Broadcast(data)
		}
	})
}
