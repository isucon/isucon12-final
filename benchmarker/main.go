package main

import (
	"context"
	"flag"
	"math/rand"
	"os"
	"time"

	"github.com/isucon/isucandar"
	"github.com/isucon/isucandar/failure"
	"github.com/isucon/isucandar/score"
	"github.com/isucon/isucon12-final/benchmarker/data"
	"github.com/isucon/isucon12-portal/bench-tool.go/benchrun"
	isuxportalResources "github.com/isucon/isucon12-portal/proto.go/isuxportal/resources"
)

type report struct {
	score    *scoreSummary
	passed   bool
	language string
}

func makeReport(scoreSummary *scoreSummary, scenario *Scenario, passed bool) *report {
	return &report{
		score:    scoreSummary,
		passed:   passed,
		language: scenario.Language,
	}
}

type scoreSummary struct {
	total     int64
	addition  int64
	deduction int64
	breakdown score.ScoreTable
}

func (r *report) toBenchmarkResult() *isuxportalResources.BenchmarkResult {
	return &isuxportalResources.BenchmarkResult{
		Finished: true,
		SurveyResponse: &isuxportalResources.SurveyResponse{
			Language: r.language,
		},
		Score: r.score.total,
		ScoreBreakdown: &isuxportalResources.BenchmarkResult_ScoreBreakdown{
			Raw:       r.score.addition,
			Deduction: r.score.deduction,
		},
		Passed: r.passed,
		Execution: &isuxportalResources.BenchmarkResult_Execution{
			Reason: func() string {
				if r.passed {
					return "passed"
				} else {
					return "failed"
				}
			}(),
		},
	}
}

type errorSummary struct {
	initializeError []error
	scenarioError   []error
	validationError []error
	internalError   []error
	unexpectedError []error
}

func (e errorSummary) containsFatal() bool {
	return len(e.internalError) != 0 || len(e.unexpectedError) != 0 || len(e.initializeError) != 0 || len(e.validationError) != 0
}

func (e errorSummary) containsError() bool {
	return e.containsFatal() || len(e.scenarioError) != 0 || len(e.internalError) != 0 || len(e.validationError) != 0 || len(e.unexpectedError) != 0
}

func init() {
	failure.BacktraceCleaner.Add(failure.SkipGOROOT)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	time.Local = time.FixedZone("GMT", 0)

	option := Option{}
	makeFlags(&option)
	if benchrun.GetTargetAddress() != "" {
		option.TargetHost = benchrun.GetTargetAddress()
	}
	AdminLogger.Print(option)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	result, scenario := runBenchmark(ctx, option)

	// 結果の集計のために少し待つ
	result.Errors.Wait()
	time.Sleep(3 * time.Second)

	errorSummary := aggregateErrors(result)
	fatal := handleErrors(errorSummary, option.PrepareOnly)

	var score *scoreSummary
	var passed bool

	if fatal {
		// エラーハンドリングの結果、fatal 判定だった場合はここで処理を終了する
		ContestantLogger.Println("続行不可能なエラーが検出されたので、ここで処理を終了します。")
		score = &scoreSummary{
			total:     0,
			addition:  0,
			deduction: 0,
			breakdown: ConstructBreakdown(result),
		}
		passed = false
	} else {
		score = sumScore(result, errorSummary)
		passed = true
	}

	ContestantLogger.Printf("[PASSED]: %v", passed)
	ContestantLogger.Printf("[SCORE] %d (addition: %d, deduction: %d)", score.total, score.addition, score.deduction)
	AdminLogger.Printf("[SCORE] %v", score.breakdown)

	if os.Getenv("ISUXBENCH_REPORT_FD") != "" {
		report := makeReport(score, scenario, passed)

		err := sendReport(report)
		if err != nil {
			// cannot continue anymore
			ContestantLogger.Printf("レポートを送信中にエラーが発生しました。スコアを送信できませんでした。エラー: %v", err)
			AdminLogger.Printf("レポートを送信中にエラーが発生しました。スコアを送信できませんでした。エラー: %+v", err)
			os.Exit(1)
		}
	}
}

func makeFlags(option *Option) {
	flag.StringVar(&option.TargetHost, "target-host", DefaultTargetHost, "Benchmark target host with port")
	flag.DurationVar(&option.RequestTimeout, "request-timeout", DefaultRequestTimeout, "Default request timeout")
	flag.DurationVar(&option.InitializeRequestTimeout, "initialize-request-timeout", DefaultInitializeRequestTimeout, "Initialize request timeout")
	flag.BoolVar(&option.ExitErrorOnFail, "exit-error-on-fail", DefaultExitErrorOnFail, "Exit with error if benchmark fails")
	flag.StringVar(&option.Stage, "stage", DefaultStage, "Set stage which affects the amount of request")
	flag.IntVar(&option.Parallelism, "max-parallelism", DefaultParallelism, "Default parallelism settings for the benchmarker")
	flag.BoolVar(&option.PrepareOnly, "prepare-only", DefaultPrepareOnly, "Run the benchmarker on preparation mode")
	flag.Parse()
}

func runBenchmark(ctx context.Context, option Option) (*isucandar.BenchmarkResult, *Scenario) {
	scenario := &Scenario{
		Option:          option,
		ConsumedUserIDs: data.NewLightSet(),
	}

	benchmark, err := isucandar.NewBenchmark(
		isucandar.WithoutPanicRecover(),
		isucandar.WithLoadTimeout(LoadingDuration),
	)
	if err != nil {
		AdminLogger.Fatal(err)
	}

	benchmark.AddScenario(scenario)

	return benchmark.Start(ctx), scenario
}

func aggregateErrors(result *isucandar.BenchmarkResult) *errorSummary {
	initializeError := []error{}
	scenarioError := []error{}
	validationError := []error{}
	internalError := []error{}
	unexpectedError := []error{}

	for _, err := range result.Errors.All() {
		category := ClassifyError(err)

		switch category {
		case InitializeErr:
			initializeError = append(initializeError, err)
		case ScenarioErr:
			scenarioError = append(scenarioError, err)
		case ValidationErr:
			validationError = append(validationError, err)
		case InternalErr:
			internalError = append(internalError, err)
		case IsucandarMarked:
			continue
		default:
			unexpectedError = append(unexpectedError, err)
		}
	}

	return &errorSummary{
		initializeError,
		scenarioError,
		validationError,
		internalError,
		unexpectedError,
	}
}

// たとえば初期化処理のみ実行するモード（prepare-only mode）の際にエラーが発生した際は「続行不可能（fatal）」として判定させるが、
// fatal と判定された場合、この関数は true として値を返す。false を返す場合は、続行可能を意味する。
func handleErrors(summary *errorSummary, prepareOnly bool) bool {
	for _, err := range summary.internalError {
		ContestantLogger.Printf("[INTERNAL] %v", err)
	}
	for _, err := range summary.unexpectedError {
		ContestantLogger.Printf("[UNEXPECTED] %v\n", err)
	}
	for _, err := range summary.initializeError {
		ContestantLogger.Printf("[INITIALIZATION_ERR] %v\n", err)
	}
	for _, err := range summary.validationError {
		ContestantLogger.Printf("[VALIDATION_ERR] %v\n", err)
	}

	// シナリオエラーは数が大量になりえるので、あまりに数が膨大になった場合には件数を絞って表示する
	var printErrorWindow []error
	aboveThreshold := false
	if len(summary.scenarioError) > MaxErrors {
		printErrorWindow = summary.scenarioError[0 : MaxErrors-1]
		aboveThreshold = true
	} else {
		printErrorWindow = summary.scenarioError[0:]
	}

	for i, err := range printErrorWindow {
		ContestantLogger.Printf("ERROR[%d] %v", i, err)
	}
	if aboveThreshold {
		ContestantLogger.Printf("ベンチマークシナリオのERRORは最大%d件まで表示しています", MaxErrors)
	}

	// prepare only モードの場合は、エラーが1件でもあればエラーで終了させる
	if prepareOnly {
		if summary.containsError() {
			return true
		}
	}

	return summary.containsFatal()
}

func sumScore(result *isucandar.BenchmarkResult, errorSummary *errorSummary) *scoreSummary {
	score := result.Score
	score = MakeScoreTable(score)

	addition := score.Sum()
	deduction := int64(len(errorSummary.scenarioError) * ErrorDeduction)

	sum := addition - deduction
	if sum < 0 {
		sum = 0
	}

	return &scoreSummary{
		total:     sum,
		addition:  addition,
		deduction: deduction,
		breakdown: ConstructBreakdown(result),
	}
}

func sendReport(report *report) error {
	benchmarkResult := report.toBenchmarkResult()

	r, err := benchrun.NewReporter(true)
	if err != nil {
		return err
	}
	if err := r.Report(benchmarkResult); err != nil {
		return err
	}

	return nil
}
