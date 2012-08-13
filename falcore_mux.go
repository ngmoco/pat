package pat

import (
	"github.com/ngmoco/falcore"
	"net/http"
	"net/url"
)
// This is 99% cut and paste from PatternServeMux so it should basically work the same way except for:
// - URL parameters are stored in the falcore.Request.Context["params"] (rather than stuffed back into the URL)
// - If nothing matches, this returns a nil Response so the next pipeline stage can run
type FalcorePatRouter struct {
	handlers map[string][]*falcoreHandler
	ParamsKey string
}
func NewFalcorePatRouter() *FalcorePatRouter {
	return &FalcorePatRouter{make(map[string][]*falcoreHandler), "params"}
}

func (p *FalcorePatRouter) SelectPipeline(req *falcore.Request) (pipe falcore.RequestFilter) {
	r := req.HttpRequest
	for _, ph := range p.handlers[r.Method] {
		if params, ok := ph.try(r.URL.Path); ok {
			req.Context[p.ParamsKey] = params
			return ph
		}
	}
	// falcore already has a fallback behavior for no matches,
	// which is to continue the pipeline so no default handler is needed here
	// The method not allowed behavior might be useful though; optional?
	return nil
}

// Head will register a pattern with a handler for HEAD requests.
func (p *FalcorePatRouter) Head(pat string, h falcore.RequestFilter) {
	p.Add("HEAD", pat, h)
}

// Get will register a pattern with a handler for GET requests.
// It also registers pat for HEAD requests. If this needs to be overridden, use
// Head before Get with pat.
func (p *FalcorePatRouter) Get(pat string, h falcore.RequestFilter) {
	p.Add("HEAD", pat, h)
	p.Add("GET", pat, h)
}

// Post will register a pattern with a handler for POST requests.
func (p *FalcorePatRouter) Post(pat string, h falcore.RequestFilter) {
	p.Add("POST", pat, h)
}

// Put will register a pattern with a handler for PUT requests.
func (p *FalcorePatRouter) Put(pat string, h falcore.RequestFilter) {
	p.Add("PUT", pat, h)
}

// Del will register a pattern with a handler for DELETE requests.
func (p *FalcorePatRouter) Del(pat string, h falcore.RequestFilter) {
	p.Add("DELETE", pat, h)
}

// Options will register a pattern with a handler for OPTIONS requests.
func (p *FalcorePatRouter) Options(pat string, h falcore.RequestFilter) {
	p.Add("OPTIONS", pat, h)
}

// Add will register a pattern with a handler for meth requests.
func (p *FalcorePatRouter) Add(meth, pat string, h falcore.RequestFilter) {
	p.handlers[meth] = append(p.handlers[meth], &falcoreHandler{pat, h})

	n := len(pat)
	if n > 0 && pat[n-1] == '/' {
		p.Add(meth, pat[:n-1], redirector(pat))
	}
}

type redirector string
func (r redirector) FilterRequest(req *falcore.Request) *http.Response {
	return falcore.RedirectResponse(req.HttpRequest, string(r))
}

type falcoreHandler struct {
	pat string
	falcore.RequestFilter
}

func (ph *falcoreHandler) try(path string) (url.Values, bool) {
	p := make(url.Values)
	var i, j int
	for i < len(path) {
		switch {
		case j >= len(ph.pat):
			if ph.pat != "/" && len(ph.pat) > 0 && ph.pat[len(ph.pat)-1] == '/' {
				return p, true
			}
			return nil, false
		case ph.pat[j] == ':':
			var name, val string
			var nextc byte
			name, nextc, j = match(ph.pat, isAlnum, j+1)
			val, _, i = match(path, matchPart(nextc), i)
			p.Add(":"+name, val)
		case path[i] == ph.pat[j]:
			i++
			j++
		default:
			return nil, false
		}
	}
	if j != len(ph.pat) {
		return nil, false
	}
	return p, true
}
