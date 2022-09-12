package main

// エラー関係をまとめて扱うファイル。

import (
	"context"
	"strings"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
)

// ISUCON 関係者向け: 細かい仕様は下記ドキュメントにまとまっている。
// https://scrapbox.io/ISUCON12/%E6%9C%AC%E9%81%B8:_%E3%82%A8%E3%83%A9%E3%83%BC%E5%91%A8%E3%82%8A%E3%81%AE%E9%9B%86%E8%A8%88%E3%81%A8%E5%AE%9F%E8%A3%85
const (
	// 初期化時のエラー。
	ErrFailedToLoadJson      failure.StringCode = "initialize-error-failed-to-load-json"
	ErrFailedInitialization  failure.StringCode = "initialize-error-failed"
	ErrPrepareInvalidRequest failure.StringCode = "initialize-error-invalid-req"

	// シナリオ中に発生するエラー。
	ErrInvalidStatusCode failure.StringCode = "scenario-error-status-code"
	ErrInvalidRequest    failure.StringCode = "scenario-error-invalid-request"
	ErrInvalidJson       failure.StringCode = "scenario-error-invalid-json"
	ErrInvalidResponse   failure.StringCode = "scenario-error-invalid-response"

	// 整合性チェック中に発生するエラー
	ValidationErrFailedToLoadJson    failure.StringCode = "validation-error-failed-to-load-json"
	ValidationErrInvalidRequest      failure.StringCode = "validation-error-invalid-request"
	ValidationErrInvalidStatusCode   failure.StringCode = "validation-error-invalid-status-code"
	ValidationErrInvalidResponseBody failure.StringCode = "validation-error-invalid-response-body"

	// ベンチマーカー内部で発生するエラー。
	ErrCannotCreateNewAgent       failure.StringCode = "internal-error-creating-agent"
	ErrCannotRefreshMasterVersion failure.StringCode = "internal-error-cannot-refresh-master"
)

type ErrCategory uint

const (
	UnexpectedErr ErrCategory = iota + 1
	InitializeErr
	ScenarioErr
	ValidationErr
	InternalErr
	IsucandarMarked
)

func ClassifyError(err error) ErrCategory {
	errCategory := UnexpectedErr

	for _, errCode := range failure.GetErrorCodes(err) {
		if IsInitializeError(errCode) {
			errCategory = InitializeErr
		} else if IsScenarioError(errCode) {
			errCategory = ScenarioErr
		} else if IsValidationError(errCode) {
			errCategory = ValidationErr
		} else if IsInternalError(errCode) {
			errCategory = InternalErr
		} else if IsIsucandarMarkedError(errCode) {
			errCategory = IsucandarMarked
		}
	}

	return errCategory
}

func IsInitializeError(code string) bool {
	return strings.HasPrefix(code, "initialize-error-")
}

func IsScenarioError(code string) bool {
	return strings.HasPrefix(code, "scenario-error-")
}

func IsValidationError(code string) bool {
	return strings.HasPrefix(code, "validation-error-")
}

func IsInternalError(code string) bool {
	return strings.HasPrefix(code, "internal-error-")
}

// isucandar が窃盗時に下記のようなプレフィックスを付与するケースがあり、正しくハンドリングできない。
// 後続の処理でスキップ等を行わせるために入れる。 FIXME ちゃんと isucandar の実装を読んで実装を直す。
func IsIsucandarMarkedError(code string) bool {
	return strings.HasPrefix(code, "prepare") || strings.HasPrefix(code, "load") || strings.HasPrefix(code, "validation")
}

func AddErrorIfNotCanceled(step *isucandar.BenchmarkStep, err error) {
	if failure.Is(err, context.Canceled) || failure.Is(err, context.DeadlineExceeded) {
		return
	}
	step.AddError(err)
}
