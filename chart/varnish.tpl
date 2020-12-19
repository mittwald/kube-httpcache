vcl 4.0;

import std;
import directors;

// ".Frontends" is a slice that contains all known Varnish instances
// (as selected by the service specified by -frontend-service).
// The backend name needs to be the Pod name, since this value is compared
// to the server identity ("server.identity" [1]) later.
//
//   [1]: https://varnish-cache.org/docs/6.4/reference/vcl.html#local-server-remote-and-client
{{ range .Frontends }}
backend {{ .Name }} {
    .host = "{{ .Host }}";
    .port = "{{ .Port }}";
}
{{- end }}

{{ range .Backends }}
backend be-{{ .Name }} {
    .host = "{{ .Host }}";
    .port = "{{ .Port }}";
}
{{- end }}

sub vcl_init {
    new cluster = directors.hash();

    {{ range .Frontends -}}
    cluster.add_backend({{ .Name }}, 1);
    {{ end }}

    new lb = directors.round_robin();

    {{ range .Backends -}}
    lb.add_backend(be-{{ .Name }});
    {{ end }}
}

sub vcl_recv
{
    # Set backend hint for non cachable objects.
    set req.backend_hint = lb.backend();
    set req.backend_hint = lb.backend();

    # ...

    # Routing logic. Pass a request to an appropriate Varnish node.
    # See https://info.varnish-software.com/blog/creating-self-routing-varnish-cluster for more info.
    unset req.http.x-cache;
    set req.backend_hint = cluster.backend(req.url);
    set req.http.x-shard = req.backend_hint;
    if (req.http.x-shard != server.identity) {
        return(pass);
    }
    set req.backend_hint = lb.backend();

    # ...

    return(hash);
}
