// MIT License
//
// Copyright (c) 2016 Rick Beton
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package afero2

import (
	"os"
	"time"

	"github.com/spf13/afero"
)

// AferoAdapter patches afero.MemMapFS to accept paths & names without a leading '/', i.e.
// to behave like the fs.FS api.
type AferoAdapter struct {
	Inner afero.Fs
}

func addLeadingSlash(name string) string {
	if len(name) > 0 && name[0] != '/' {
		name = "/" + name
	}
	return name
}

func (aa AferoAdapter) Create(name string) (afero.File, error) {
	return aa.Inner.Create(addLeadingSlash(name))
}

func (aa AferoAdapter) Mkdir(name string, perm os.FileMode) error {
	return aa.Inner.Mkdir(addLeadingSlash(name), perm)
}

func (aa AferoAdapter) MkdirAll(path string, perm os.FileMode) error {
	return aa.Inner.MkdirAll(addLeadingSlash(path), perm)
}

func (aa AferoAdapter) Open(name string) (afero.File, error) {
	return aa.Inner.Open(addLeadingSlash(name))
}

func (aa AferoAdapter) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return aa.Inner.OpenFile(addLeadingSlash(name), flag, perm)
}

func (aa AferoAdapter) Remove(name string) error {
	return aa.Inner.Remove(addLeadingSlash(name))
}

func (aa AferoAdapter) RemoveAll(path string) error {
	return aa.Inner.RemoveAll(addLeadingSlash(path))
}

func (aa AferoAdapter) Rename(oldname, newname string) error {
	return aa.Inner.Rename(addLeadingSlash(oldname), addLeadingSlash(newname))
}

func (aa AferoAdapter) Stat(name string) (os.FileInfo, error) {
	return aa.Inner.Stat(addLeadingSlash(name))
}

func (aa AferoAdapter) Name() string {
	return aa.Inner.Name()
}

func (aa AferoAdapter) Chmod(name string, mode os.FileMode) error {
	return aa.Inner.Chmod(addLeadingSlash(name), mode)
}

func (aa AferoAdapter) Chown(name string, uid, gid int) error {
	return aa.Inner.Chown(addLeadingSlash(name), uid, gid)
}

func (aa AferoAdapter) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return aa.Inner.Chtimes(addLeadingSlash(name), atime, mtime)
}
