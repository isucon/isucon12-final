package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
)

type ValidationError struct {
	Errors   []error
	Canceled bool
}

func (v ValidationError) Error() string {
	messages := []string{}

	for _, err := range v.Errors {
		if err != nil {
			messages = append(messages, fmt.Sprintf("%v", err))
		}
	}

	return strings.Join(messages, "\n")
}

func (v ValidationError) IsEmpty() bool {
	if v.Canceled {
		// ベンチマーカーがキャンセル済みの場合、空だったとしても false で返す
		// 得点を追加させないために必要
		return false
	}
	for _, err := range v.Errors {
		if err != nil {
			if ve, ok := err.(ValidationError); ok {
				if !ve.IsEmpty() {
					return false
				}
			} else {
				return false
			}
		}
	}
	return true
}

func (v ValidationError) Add(step *isucandar.BenchmarkStep) {
	for _, err := range v.Errors {
		if err != nil {
			if ve, ok := err.(ValidationError); ok {
				ve.Add(step)
			} else {
				step.AddError(err)
			}
		}
	}
}

type ResponseValidator func(*http.Response) error

func ValidateResponse(res *http.Response, validators ...ResponseValidator) ValidationError {
	defer io.Copy(io.Discard, res.Body) // nolint:errcheck
	errs := []error{}
	canceled := false

	for _, validator := range validators {
		err := validator(res)
		if err != nil {
			// ベンチが終了したタイミングでは、タイムアウトあるいはキャンセルになる。
			// ベンチが終了したタイミングではエラーを追加しないようにしている。
			if failure.Is(err, context.DeadlineExceeded) || failure.Is(err, context.Canceled) {
				canceled = true
				continue
			}
			errs = append(errs, err)
		}
	}

	return ValidationError{
		Errors:   errs,
		Canceled: canceled,
	}
}

func WithStatusCode(statusCode int) ResponseValidator {
	return func(r *http.Response) error {
		if r.StatusCode != statusCode {
			return failure.NewError(
				ErrInvalidStatusCode,
				fmt.Errorf(
					"%s %s : expected(%d) != actual(%d)",
					r.Request.Method,
					r.Request.URL.Path,
					statusCode,
					r.StatusCode,
				),
			)
		}
		return nil
	}
}

func WithInitializationSuccess[T any](target *T) ResponseValidator {
	return func(r *http.Response) error {
		if r.StatusCode != http.StatusOK {
			return failure.NewError(
				ErrFailedInitialization,
				fmt.Errorf(
					"%s %s : expected(%d) != actual(%d)",
					r.Request.Method,
					r.Request.URL.Path,
					http.StatusOK,
					r.StatusCode,
				),
			)
		}

		if err := parseJsonBody(r, target); err != nil {
			return failure.NewError(
				ErrFailedInitialization,
				err,
			)
		}

		return nil
	}
}

// パース可能な JSON ボディをレスポンスにもっているかを確認する。
// パース可能かどうかだけを見るので、その中の値を見ていない点に注意が必要。
func WithJsonBody[T any](target *T) ResponseValidator {
	return func(r *http.Response) error {
		return parseJsonBody(r, target)
	}
}

type ValidationHint struct {
	endpoint string
	what     string
}

func Hint(endpoint, what string) func() ValidationHint {
	return func() ValidationHint {
		return ValidationHint{
			endpoint,
			what,
		}
	}
}

func makeInconsistentMsg[T any](expected, actual T, b func() ValidationHint) error {
	return fmt.Errorf(
		"%v : expected(%v) != actual(%v)", fmt.Sprintf("%s の Body の %s が違います", b().endpoint, b().what), expected, actual,
	)
}

func makeInconsistentStatusCode(expected, actual int, b func() ValidationHint) error {
	return fmt.Errorf(
		"%v : expected(%v) != actual(%v)", fmt.Sprintf("%s の Response の　HTTP Status Code が違います", b().endpoint), expected, actual,
	)
}

func IsuAssert[T comparable](expected, actual T, b func() ValidationHint) error {
	if expected != actual {

		return failure.NewError(
			ValidationErrInvalidResponseBody,
			makeInconsistentMsg(expected, actual, b),
		)
	}
	return nil
}

func IsuAssertStatus(expected, actual int, b func() ValidationHint) error {
	if expected != actual {

		return failure.NewError(
			ValidationErrInvalidStatusCode,
			makeInconsistentStatusCode(expected, actual, b),
		)
	}
	return nil
}

func IgnoreWhat(name ...string) []string {
	var names []string
	return append(names, name...)
}

func Diff[T comparable](expected, actual T, b func() ValidationHint, ignoreWhats ...string) error {
	expectedRefl := reflect.ValueOf(expected).Elem()
	actualRefl := reflect.ValueOf(actual).Elem()
	expectedStruct := reflect.TypeOf(expected).Elem()

	for i := 0; i < expectedRefl.NumField(); i++ {
		expectedField := expectedRefl.Field(i)
		actualField := actualRefl.Field(i)
		expectedWhat := strings.Split(expectedStruct.Field(i).Tag.Get("json"), ",")[0] //,omitemptyを除外した文字列
		expectedKind := expectedStruct.Field(i).Type.Kind()

		isCheck := true
		for _, v := range ignoreWhats {
			if v == expectedWhat {
				isCheck = false
				break
			}
		}

		// チェックする必要があるstruct fieldか確認
		if !isCheck {
			continue
		}

		nextB := Hint(b().endpoint, b().what+expectedWhat)

		if err := IsuTypeAssert(expectedKind, expectedField, actualField, nextB); err != nil {
			// エラーをすぐに返したくない場合は、リストを外から渡すなどする。
			return err
		}
	}

	return nil
}

func IsuTypeAssert(expectedKind reflect.Kind, expected, actual reflect.Value, b func() ValidationHint) error {

	if !IsuCompare(expectedKind, expected, actual) {
		return failure.NewError(
			ValidationErrInvalidResponseBody,
			makeInconsistentMsg(expected, actual, b),
		)
	}
	return nil
}

func IsuCompare(expectedKind reflect.Kind, expected, actual reflect.Value) bool {
	switch expectedKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return expected.Int() == actual.Int()
	case reflect.Bool:
		return expected.Bool() == actual.Bool()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return expected.Uint() == actual.Uint()
	case reflect.Float32, reflect.Float64:
		return expected.Float() == actual.Float()
	case reflect.String:
		return expected.String() == actual.String()
	case reflect.Ptr:
		if expected.IsNil() && actual.IsNil() {
			return true
		}
		return &expected == &actual

	default:
		panic(fmt.Sprintf("%v kind not handled", expectedKind))
	}
}
