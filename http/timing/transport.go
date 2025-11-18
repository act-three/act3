package timing

import "net/http"

type transport struct {
	metric string
	rt     http.RoundTripper // if nil, uses http.DefaultTransport
}

// Transport records the duration of each round trip
// in the request's context under the given metric.
func Transport(metric string, rt http.RoundTripper) http.RoundTripper {
	return &transport{metric: metric, rt: rt}
}

func (t *transport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	rt := t.rt
	if rt == nil {
		rt = http.DefaultTransport
	}
	Measure(req.Context(), t.metric, func() {
		resp, err = rt.RoundTrip(req)
	})
	return resp, err
}
