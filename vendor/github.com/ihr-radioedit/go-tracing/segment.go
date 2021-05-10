package tracing

import "net/http"

type Segment interface {
	SetAttribute(tag string, val interface{}) // add key/value data to the segment
	Finish(err error)                   // call this method to finish the segment, err may be nil
}

type ExtSegment interface {
	Segment
	SetResponse(r *http.Response)
}

type multiSegment []Segment

func (s multiSegment) SetAttribute(tag string, val interface{}) {
	for _, v := range s {
		v.SetAttribute(tag, val)
	}
}

func (s multiSegment) Finish(err error) {
	for _, v := range s {
		v.Finish(err)
	}
}

type multiExtSegment []ExtSegment

func (s multiExtSegment) SetAttribute(tag string, val interface{}) {
	for _, v := range s {
		v.SetAttribute(tag, val)
	}
}

func (s multiExtSegment) Finish(err error) {
	for _, v := range s {
		v.Finish(err)
	}
}

func (s multiExtSegment) SetResponse(r *http.Response) {
	for _, v := range s {
		v.SetResponse(r)
	}
}

type segment struct {
	setAttribute func(key string, val interface{})
	setResponse func(r *http.Response)
	finish func(err error)
}

func (s segment) SetAttribute(key string, val interface{}) {
	s.setAttribute(key, val)
}

func (s segment) SetResponse(r *http.Response) {
	s.setResponse(r)
}

func (s segment) Finish(err error) {
	s.finish(err)
}
