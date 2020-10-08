package servefiles

import "strings"

type commaSeparatedList string

func (list commaSeparatedList) Contains(want string) bool {
	accepted := strings.Split(string(list), ",")
	for _, encoding := range accepted {
		if strings.TrimSpace(encoding) == want {
			return true
		}
	}
	return false
}

//-------------------------------------------------------------------------------------------------

type code int

const (
	Directory code = 0
	Continue  code = 100
	//OK                 code = 200
	//NotModified        code = 304
	Forbidden          code = 403
	NotFound           code = 404
	ServiceUnavailable code = 503
)

func (code code) String() string {
	switch code {
	case Continue:
		return "100 Continue"
	//case OK:
	//	return "200 OK"
	//case NotModified:
	//	return "304 Not modified"
	case Forbidden:
		return "403 Forbidden"
	case NotFound:
		return "404 Not found"
	case ServiceUnavailable:
		return "503 Service unavailable"
	}
	panic(code)
}
