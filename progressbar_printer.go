package pterm

import (
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"sync"
	"time"

	"atomicgo.dev/cursor"
	"atomicgo.dev/schedule"
	"go.uber.org/atomic"

	"github.com/gookit/color"

	"github.com/pterm/pterm/internal"
)

// activeProgressBarPrinters contains all running ProgressbarPrinters.
// Generally, there should only be one active ProgressbarPrinter at a time.
type atomicActiveProgressBarPrinters struct {
	printers []*ProgressbarPrinter
	lock     *sync.Mutex
}

var (
	// DefaultProgressbar is the default ProgressbarPrinter.
	DefaultProgressbar = ProgressbarPrinter{
		Total:                     100,
		BarCharacter:              "█",
		LastCharacter:             "█",
		ElapsedTimeRoundingFactor: time.Second,
		BarStyle:                  &ThemeDefault.ProgressbarBarStyle,
		TitleStyle:                &ThemeDefault.ProgressbarTitleStyle,
		ShowTitle:                 true,
		ShowCount:                 true,
		ShowPercentage:            true,
		ShowElapsedTime:           true,
		BarFiller:                 Gray("█"),
		MaxWidth:                  80,
		Writer:                    os.Stdout,
	}

	activeProgressBarPrinters = atomicActiveProgressBarPrinters{
		printers: []*ProgressbarPrinter{},
		lock:     &sync.Mutex{},
	}
)

// ProgressbarPrinter shows a progress animation in the terminal.
type ProgressbarPrinter struct {
	Title                     string
	Total                     int
	Current                   int
	BarCharacter              string
	LastCharacter             string
	ElapsedTimeRoundingFactor time.Duration
	BarFiller                 string
	MaxWidth                  int

	ShowElapsedTime bool
	ShowCount       bool
	ShowTitle       bool
	ShowPercentage  bool
	RemoveWhenDone  bool

	TitleStyle *Style
	BarStyle   *Style

	IsActive bool

	startedAt    time.Time
	rerenderTask *schedule.Task

	Writer io.Writer

	// Thread-safe versions of existing variables used internally
	atomicIsActive *atomic.Bool
	atomicTitle    *atomic.String
}

// Lazy init used to initialize thread-safe variables
func (p *ProgressbarPrinter) lazyInit() {
	if p.atomicIsActive == nil {
		p.atomicIsActive = atomic.NewBool(p.IsActive)
	}
	if p.atomicTitle == nil {
		p.atomicTitle = atomic.NewString(p.Title)
	}
}

// WithTitle sets the name of the ProgressbarPrinter.
func (p ProgressbarPrinter) WithTitle(name string) *ProgressbarPrinter {
	p.lazyInit()
	p.atomicTitle.Store(name)
	// We still set Title here so it is available to the users, it is not read anywhere
	p.Title = name
	return &p
}

// WithMaxWidth sets the maximum width of the ProgressbarPrinter.
// If the terminal is smaller than the given width, the terminal width will be used instead.
// If the width is set to zero, or below, the terminal width will be used.
func (p ProgressbarPrinter) WithMaxWidth(maxWidth int) *ProgressbarPrinter {
	p.lazyInit()
	p.MaxWidth = maxWidth
	return &p
}

// WithTotal sets the total value of the ProgressbarPrinter.
func (p ProgressbarPrinter) WithTotal(total int) *ProgressbarPrinter {
	p.lazyInit()
	p.Total = total
	return &p
}

// WithCurrent sets the current value of the ProgressbarPrinter.
func (p ProgressbarPrinter) WithCurrent(current int) *ProgressbarPrinter {
	p.lazyInit()
	p.Current = current
	return &p
}

// WithBarCharacter sets the bar character of the ProgressbarPrinter.
func (p ProgressbarPrinter) WithBarCharacter(char string) *ProgressbarPrinter {
	p.lazyInit()
	p.BarCharacter = char
	return &p
}

// WithLastCharacter sets the last character of the ProgressbarPrinter.
func (p ProgressbarPrinter) WithLastCharacter(char string) *ProgressbarPrinter {
	p.lazyInit()
	p.LastCharacter = char
	return &p
}

// WithElapsedTimeRoundingFactor sets the rounding factor of the elapsed time.
func (p ProgressbarPrinter) WithElapsedTimeRoundingFactor(duration time.Duration) *ProgressbarPrinter {
	p.lazyInit()
	p.ElapsedTimeRoundingFactor = duration
	return &p
}

// WithShowElapsedTime sets if the elapsed time should be displayed in the ProgressbarPrinter.
func (p ProgressbarPrinter) WithShowElapsedTime(b ...bool) *ProgressbarPrinter {
	p.lazyInit()
	p.ShowElapsedTime = internal.WithBoolean(b)
	return &p
}

// WithShowCount sets if the total and current count should be displayed in the ProgressbarPrinter.
func (p ProgressbarPrinter) WithShowCount(b ...bool) *ProgressbarPrinter {
	p.lazyInit()
	p.ShowCount = internal.WithBoolean(b)
	return &p
}

// WithShowTitle sets if the title should be displayed in the ProgressbarPrinter.
func (p ProgressbarPrinter) WithShowTitle(b ...bool) *ProgressbarPrinter {
	p.lazyInit()
	p.ShowTitle = internal.WithBoolean(b)
	return &p
}

// WithShowPercentage sets if the completed percentage should be displayed in the ProgressbarPrinter.
func (p ProgressbarPrinter) WithShowPercentage(b ...bool) *ProgressbarPrinter {
	p.lazyInit()
	p.ShowPercentage = internal.WithBoolean(b)
	return &p
}

// WithStartedAt sets the time when the ProgressbarPrinter started.
func (p ProgressbarPrinter) WithStartedAt(t time.Time) *ProgressbarPrinter {
	p.lazyInit()
	p.startedAt = t
	return &p
}

// WithTitleStyle sets the style of the title.
func (p ProgressbarPrinter) WithTitleStyle(style *Style) *ProgressbarPrinter {
	p.lazyInit()
	p.TitleStyle = style
	return &p
}

// WithBarStyle sets the style of the bar.
func (p ProgressbarPrinter) WithBarStyle(style *Style) *ProgressbarPrinter {
	p.lazyInit()
	p.BarStyle = style
	return &p
}

// WithRemoveWhenDone sets if the ProgressbarPrinter should be removed when it is done.
func (p ProgressbarPrinter) WithRemoveWhenDone(b ...bool) *ProgressbarPrinter {
	p.lazyInit()
	p.RemoveWhenDone = internal.WithBoolean(b)
	return &p
}

// WithBarFiller sets the filler character for the ProgressbarPrinter.
func (p ProgressbarPrinter) WithBarFiller(char string) *ProgressbarPrinter {
	p.lazyInit()
	p.BarFiller = char
	return &p
}

// WithWriter sets the custom Writer.
func (p ProgressbarPrinter) WithWriter(writer io.Writer) *ProgressbarPrinter {
	p.lazyInit()
	p.Writer = writer
	return &p
}

// SetWriter sets the custom Writer.
func (p *ProgressbarPrinter) SetWriter(writer io.Writer) {
	p.Writer = writer
}

// SetStartedAt sets the time when the ProgressbarPrinter started.
func (p *ProgressbarPrinter) SetStartedAt(t time.Time) {
	p.startedAt = t
}

// ResetTimer resets the timer of the ProgressbarPrinter.
func (p *ProgressbarPrinter) ResetTimer() {
	p.startedAt = time.Now()
}

// Increment current value by one.
func (p *ProgressbarPrinter) Increment() *ProgressbarPrinter {
	p.Add(1)
	return p
}

// UpdateTitle updates the title and re-renders the progressbar
func (p *ProgressbarPrinter) UpdateTitle(title string) *ProgressbarPrinter {
	p.atomicTitle.Store(title)
	// We still set Title here so it is available to the users, it is not read anywhere
	p.Title = title
	p.updateProgress()
	return p
}

// This is the update logic, renders the progressbar
func (p *ProgressbarPrinter) updateProgress() *ProgressbarPrinter {
	Fprinto(p.Writer, p.getString())
	return p
}

func (p *ProgressbarPrinter) getString() string {
	if !p.atomicIsActive.Load() {
		return ""
	}
	if p.TitleStyle == nil {
		p.TitleStyle = NewStyle()
	}
	if p.BarStyle == nil {
		p.BarStyle = NewStyle()
	}
	if p.Total == 0 {
		return ""
	}

	var before string
	var after string
	var width int

	if p.MaxWidth <= 0 {
		width = GetTerminalWidth()
	} else if GetTerminalWidth() < p.MaxWidth {
		width = GetTerminalWidth()
	} else {
		width = p.MaxWidth
	}

	if p.ShowTitle {
		before += p.TitleStyle.Sprint(p.atomicTitle.Load()) + " "
	}
	if p.ShowCount {
		padding := 1 + int(math.Log10(float64(p.Total)))
		before += Gray("[") + LightWhite(fmt.Sprintf("%0*d", padding, p.Current)) + Gray("/") + LightWhite(p.Total) + Gray("]") + " "
	}

	after += " "

	if p.ShowPercentage {
		currentPercentage := int(internal.PercentageRound(float64(int64(p.Total)), float64(int64(p.Current))))
		decoratorCurrentPercentage := color.RGB(NewRGB(255, 0, 0).Fade(0, float32(p.Total), float32(p.Current), NewRGB(0, 255, 0)).GetValues()).
			Sprintf("%3d%%", currentPercentage)
		after += decoratorCurrentPercentage + " "
	}
	if p.ShowElapsedTime {
		after += "| " + p.parseElapsedTime()
	}

	barMaxLength := width - len(RemoveColorFromString(before)) - len(RemoveColorFromString(after)) - 1

	barCurrentLength := (p.Current * barMaxLength) / p.Total
	var barFiller string
	if barMaxLength-barCurrentLength > 0 {
		barFiller = strings.Repeat(p.BarFiller, barMaxLength-barCurrentLength)
	}

	bar := barFiller
	if barCurrentLength > 0 {
		bar = p.BarStyle.Sprint(strings.Repeat(p.BarCharacter, barCurrentLength)+p.LastCharacter) + bar
	}

	return before + bar + after
}

// Add to current value.
func (p *ProgressbarPrinter) Add(count int) *ProgressbarPrinter {
	if p.Total == 0 {
		return nil
	}

	p.Current += count
	p.updateProgress()

	if p.Current >= p.Total {
		p.Total = p.Current
		p.updateProgress()
		p.Stop()
	}
	return p
}

// Start the ProgressbarPrinter.
func (p ProgressbarPrinter) Start(title ...interface{}) (*ProgressbarPrinter, error) {
	p.lazyInit()
	cursor.Hide()
	p.IsActive = true
	p.atomicIsActive.Store(p.IsActive)
	if len(title) != 0 {
		p.Title = Sprint(title...)
		p.atomicTitle.Store(p.Title)
	}
	if RawOutput.Load() && p.ShowTitle {
		Fprintln(p.Writer, p.atomicTitle.Load())
	}

	activeProgressBarPrinters.lock.Lock()
	activeProgressBarPrinters.printers = append(activeProgressBarPrinters.printers, &p)
	activeProgressBarPrinters.lock.Unlock()

	p.startedAt = time.Now()

	p.updateProgress()

	if p.ShowElapsedTime {
		p.rerenderTask = schedule.Every(time.Second, func() bool {
			p.updateProgress()
			return true
		})
	}

	return &p, nil
}

// Stop the ProgressbarPrinter.
func (p *ProgressbarPrinter) Stop() (*ProgressbarPrinter, error) {
	if p.rerenderTask != nil && p.rerenderTask.IsActive() {
		p.rerenderTask.Stop()
	}
	cursor.Show()

	if !p.atomicIsActive.Load() {
		return p, nil
	}
	p.IsActive = false
	p.atomicIsActive.Store(false)
	if p.RemoveWhenDone {
		fClearLine(p.Writer)
		Fprinto(p.Writer)
	} else {
		Fprintln(p.Writer)
	}
	return p, nil
}

// GenericStart runs Start, but returns a LivePrinter.
// This is used for the interface LivePrinter.
// You most likely want to use Start instead of this in your program.
func (p *ProgressbarPrinter) GenericStart() (*LivePrinter, error) {
	p2, _ := p.Start()
	lp := LivePrinter(p2)
	return &lp, nil
}

// GenericStop runs Stop, but returns a LivePrinter.
// This is used for the interface LivePrinter.
// You most likely want to use Stop instead of this in your program.
func (p *ProgressbarPrinter) GenericStop() (*LivePrinter, error) {
	p2, _ := p.Stop()
	lp := LivePrinter(p2)
	return &lp, nil
}

// GetElapsedTime returns the elapsed time, since the ProgressbarPrinter was started.
func (p *ProgressbarPrinter) GetElapsedTime() time.Duration {
	return time.Since(p.startedAt)
}

func (p *ProgressbarPrinter) parseElapsedTime() string {
	s := p.GetElapsedTime().Round(p.ElapsedTimeRoundingFactor).String()
	return s
}
