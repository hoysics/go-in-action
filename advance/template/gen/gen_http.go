package template

import (
	"io"
	"text/template"
)

type ServiceDefinition struct {
	Name    string
	Methods []Method
}

func (s *ServiceDefinition) GenName() string {
	return s.Name + "Gen"
}

type Method struct {
	Name         string
	ReqTypeName  string
	RespTypeName string
}

// 这是你们的作业，你们需要补全这个 template
const serviceTpl = `
{{- $service :=.GenName -}}
type {{ $service }} struct {
    Endpoint string
    Path string
	Client http.Client
}
{{range $idx, $method := .Methods}}
func (s *{{$service}}) {{$method.Name}}(ctx context.Context, req *{{$method.ReqTypeName}}) (*{{$method.RespTypeName}}, error) {
	url := s.Endpoint + s.Path + "/{{$method.Name}}"
	var bys  []byte
	var err error
	if bys, err = json.Marshal(req);err != nil {
		return nil, err
	}
	body := &bytes.Buffer{}
	body.Write(bys)
	req, err := http.NewRequest(ctx, "POST", url, body)
	if err != nil {
		return nil, err
	}
	resp, err := s.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bys, err = io.ReadAll(resp.Body)
	resp := &{{$method.RespTypeName}}{}
	err = json.Unmarshal(bys, resp)
	return resp, err
}
{{end}}
`

func Gen(writer io.Writer, def *ServiceDefinition) error {
	tpl := template.New("service")
	tpl, err := tpl.Parse(serviceTpl)
	if err != nil {
		return err
	}
	// 还可以进一步调用 format.Source 来格式化生成代码
	return tpl.Execute(writer, def)
}
