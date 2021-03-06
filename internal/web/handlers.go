package web

import (
	"io"
	"net/http"
	"os"
)

const (
	indexPath  = "./web/index.html"
	loginPath  = "./web/login.html"
	mobilePage = "./web/mobile.html"
)

// GET /
//
func (s Server) index(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(indexPath)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, f)
	if err != nil {
		s.logger.Errorf("can't io.Copy() %s: %s\n", f.Name(), err)
	}
	f.Close()
}

// GET /mobile
//
func (s Server) mobile(w http.ResponseWriter, r *http.Request) {
	f, err := os.Open(mobilePage)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, f)
	if err != nil {
		s.logger.Errorf("can't io.Copy() %s: %s\n", f.Name(), err)
	}
	f.Close()
}

// GET /login
//
func (s Server) login(w http.ResponseWriter, r *http.Request) {
	// Redirect to / if user is authorized
	c, err := r.Cookie(s.config.AuthCookieName)
	if err == nil && s.authService.CheckToken(c.Value) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}

	f, err := os.Open(loginPath)
	if err != nil {
		s.processError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, f)
	if err != nil {
		s.logger.Errorf("can't io.Copy() %s: %s\n", f.Name(), err)
	}
	f.Close()
}

// GET /version
//
// Response: backend version
//
func (s Server) backendVersion(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(s.config.Version))
}

// extensionHandler servers extensions
func (s Server) extensionHandler(dir http.Dir) http.Handler {
	const blankFilename = "_blank.png"
	const iconExt = ".png"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ext := r.URL.Path

		if f, err := dir.Open(ext + iconExt); err == nil {
			// Return existing icon
			io.Copy(w, f)
			if err != nil {
				s.logger.Errorf("can't io.Copy() %s.png: %s\n", ext, err)
			}
			f.Close()
			return
		}

		// return blank icon
		f, err := dir.Open(blankFilename)
		if err != nil {
			return
		}
		_, err = io.Copy(w, f)
		if err != nil {
			s.logger.Errorf("can't io.Copy() %s.png: %s\n", ext, err)
		}
		f.Close()
		return
	})
}

func mock(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Mock"))
}
