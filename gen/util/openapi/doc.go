package openapi

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/RussellLuo/kok/gen/util/reflector"
)

var (
	reKok = regexp.MustCompile(`@kok\((\w+)\):\s*"(.+)"`)
)

// result 提供函数名和参数详情，doc提供method pattern
func FromDoc(result *reflector.Result, doc map[string][]string) (*Specification, error) {
	spec := &Specification{}

	for _, m := range result.Interface.Methods {
		comments, ok := doc[m.Name]
		if !ok {
			continue
		}

		op := &Operation{Name: m.Name}

		// Add all request parameters with specified Name/Type
		params := make(map[string]*Param)
		for _, mp := range m.Params {
			p := &Param{
				In:   InBody, // param is in body by default
				Type: mp.Type,
			}
			p.SetName(mp.Name)
			op.addParam(p)

			// Build the mapping for later manipulation.
			params[p.Name] = p
		}
		// 从接口定义中获得每个函数的所有参数定义, 一个函数对应一个operation和params

		// Set a default success response.
		op.Resp(http.StatusOK, MediaTypeJSON, nil)

		// 注释填充operation
		if err := manipulateByComments(op, params, comments); err != nil {
			return nil, err
		}

		spec.Operations = append(spec.Operations, op)
	}

	return spec, nil
}

// 注释填充operation
// params + comment -> operation
func manipulateByComments(op *Operation, params map[string]*Param, comments []string) error {
	for _, comment := range comments {
		if !strings.Contains(comment, "@kok") {
			continue
		}

		result := reKok.FindStringSubmatch(comment)
		if len(result) != 3 {
			return fmt.Errorf("invalid kok comment: %s", comment)
		}

		// 把注释split开
		key, value := result[1], result[2]
		switch key {
		case "op":
			fields := strings.Fields(value)
			if len(fields) != 2 {
				return fmt.Errorf(`%q does not match the expected format: "<METHOD> <PATH>"`, value)
			}
			op.Method, op.Pattern = fields[0], fields[1]
		case "param":
			p := op.buildParam(value, "", "") // no default name and type

			name, subName := splitParamName(p.Name)
			param, ok := params[name]
			if !ok {
				return fmt.Errorf("no param `%s` declared in the method %s", name, op.Name)
			}

			if subName == "" {
				param.Set(p)
			} else {
				p.SetName(subName)
				param.Add(p)
			}
		case "success":
			op.SuccessResponse, op.Options.ResponseEncoder.Success = buildSuccessResponse(value)
		case "failure":
			op.Options.ResponseEncoder.Failure = getFailureResponseEncoder(value)
		default:
			return fmt.Errorf(`unrecognized kok key "%s" in comment: %s`, key, comment)
		}
	}

	if op.Method == "" {
		return fmt.Errorf("method %s has no comment about @kok(method)", op.Name)
	}

	if op.Pattern == "" {
		return fmt.Errorf("method %s has no comment about @kok(pattern)", op.Name)
	}

	return nil
}
