package gen

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/RussellLuo/kok/gen/endpoint"
	"github.com/RussellLuo/kok/gen/http/chi"
	"github.com/RussellLuo/kok/gen/http/httpclient"
	"github.com/RussellLuo/kok/gen/http/httptest"
	"github.com/RussellLuo/kok/gen/util/openapi"
	"github.com/RussellLuo/kok/gen/util/reflector"
)

type Options struct {
	SchemaPtr         bool
	SchemaTag         string
	TagKeyToSnakeCase bool
	Formatted         bool
	EnableTracing     bool
}

type Content struct {
	Endpoint   []byte
	HTTP       []byte
	HTTPTest   []byte
	HTTPClient []byte
}

type Generator struct {
	endpoint   *endpoint.Generator
	chi        *chi.Generator
	httptest   *httptest.Generator
	httpclient *httpclient.Generator
}

func New(opts Options) *Generator {
	return &Generator{
		endpoint: endpoint.New(&endpoint.Options{
			SchemaPtr:         opts.SchemaPtr,
			SchemaTag:         opts.SchemaTag,
			TagKeyToSnakeCase: opts.TagKeyToSnakeCase,
			Formatted:         opts.Formatted,
		}),
		chi: chi.New(&chi.Options{
			SchemaPtr:         opts.SchemaPtr,
			SchemaTag:         opts.SchemaTag,
			TagKeyToSnakeCase: opts.TagKeyToSnakeCase,
			Formatted:         opts.Formatted,
			EnableTracing:     opts.EnableTracing,
		}),
		httptest: httptest.New(&httptest.Options{
			Formatted: opts.Formatted,
		}),
		httpclient: httpclient.New(&httpclient.Options{
			SchemaPtr:         opts.SchemaPtr,
			SchemaTag:         opts.SchemaTag,
			TagKeyToSnakeCase: opts.TagKeyToSnakeCase,
			Formatted:         opts.Formatted,
		}),
	}
}

func (g *Generator) Generate(srcFilename, interfaceName, dstPkgName, testFilename string) (content Content, err error) {
	// 从源文件获取interface obj
	// 构造自定义的interfaceType
	result, err := reflector.ReflectInterface(filepath.Dir(srcFilename), dstPkgName, interfaceName)
	if err != nil {
		return content, err
	}

	// 拿到所有注释, map["func"] = "POST /xxx" -> operations， 这就是结构化的注释
	spec, err := g.getSpec(result, srcFilename, interfaceName) // 生成openapi的struct
	if err != nil {
		return content, err
	}

	// Generate the endpoint code.
	// 利用go template进行文本替换
	content.Endpoint, err = g.endpoint.Generate(result, spec)
	if err != nil {
		return content, err
	}

	// Generate the HTTP code.
	content.HTTP, err = g.chi.Generate(result, spec)
	if err != nil {
		return content, err
	}

	// Generate the HTTP client code.
	content.HTTPClient, err = g.httpclient.Generate(result, spec)
	if err != nil {
		return content, err
	}

	// Generate the HTTP tests code.
	content.HTTPTest, err = g.httptest.Generate(result, testFilename)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("WARNING: Skip generating the HTTP tests due to an error (%v)\n", err)
			return content, nil
		}
		return content, err
	}

	return content, nil
}

// parse file -> ast.File -> interfaceType -> doc map -> operations
func (g *Generator) getSpec(result *reflector.Result, srcFilename, interfaceName string) (*openapi.Specification, error) {
	// 获得原始注释
	// parse源文件生成ast.File->interfaceType->map[funcName] = "doc"
	doc, err := reflector.GetInterfaceMethodDoc(srcFilename, interfaceName) // 收集interface的注释 map[methodName] = comment
	if err != nil {
		return nil, err
	}
	fmt.Println(doc)

	// doc 构造为operation
	spec, err := openapi.FromDoc(result, doc)
	if err != nil {
		return nil, err
	}

	return spec, nil
}
