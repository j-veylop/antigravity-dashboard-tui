package models

import (
	"testing"
)

func TestTimeRange_String(t *testing.T) {
	tests := []struct {
		name string
		tr   TimeRange
		want string
	}{
		{"24Hours", TimeRange24Hours, "24 Hours"},
		{"7Days", TimeRange7Days, "7 Days"},
		{"30Days", TimeRange30Days, "30 Days"},
		{"AllTime", TimeRangeAllTime, "All Time"},
		{"Unknown", TimeRange(999), "Unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.String(); got != tt.want {
				t.Errorf("TimeRange.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeRange_Days(t *testing.T) {
	tests := []struct {
		name string
		tr   TimeRange
		want int
	}{
		{"24Hours", TimeRange24Hours, 1},
		{"7Days", TimeRange7Days, 7},
		{"30Days", TimeRange30Days, 30},
		{"AllTime", TimeRangeAllTime, 0},
		{"Unknown", TimeRange(999), 30},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.Days(); got != tt.want {
				t.Errorf("TimeRange.Days() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeRange_Next(t *testing.T) {
	tests := []struct {
		name string
		tr   TimeRange
		want TimeRange
	}{
		{"24Hours -> 7Days", TimeRange24Hours, TimeRange7Days},
		{"7Days -> 30Days", TimeRange7Days, TimeRange30Days},
		{"30Days -> AllTime", TimeRange30Days, TimeRangeAllTime},
		{"AllTime -> 24Hours", TimeRangeAllTime, TimeRange24Hours},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.tr.Next(); got != tt.want {
				t.Errorf("TimeRange.Next() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccountHistoryStats_HasData(t *testing.T) {
	tests := []struct {
		name  string
		stats AccountHistoryStats
		want  bool
	}{
		{"NoData", AccountHistoryStats{TotalDataPoints: 0}, false},
		{"HasData", AccountHistoryStats{TotalDataPoints: 5}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.stats.HasData(); got != tt.want {
				t.Errorf("AccountHistoryStats.HasData() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAccountHistoryStats_GetPeakHour(t *testing.T) {
	tests := []struct {
		name         string
		patterns     []HourlyPattern
		wantPeakHour int
		wantPeakVal  float64
	}{
		{
			name:         "Empty",
			patterns:     nil,
			wantPeakHour: 0,
			wantPeakVal:  0,
		},
		{
			name: "SinglePeak",
			patterns: []HourlyPattern{
				{Hour: 10, AvgConsumed: 50.5},
				{Hour: 11, AvgConsumed: 20.0},
			},
			wantPeakHour: 10,
			wantPeakVal:  50.5,
		},
		{
			name: "MultiplePeaks",
			patterns: []HourlyPattern{
				{Hour: 1, AvgConsumed: 10.0},
				{Hour: 15, AvgConsumed: 100.0},
				{Hour: 20, AvgConsumed: 50.0},
			},
			wantPeakHour: 15,
			wantPeakVal:  100.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AccountHistoryStats{HourlyPatterns: tt.patterns}
			gotHour, gotVal := a.GetPeakHour()
			if gotHour != tt.wantPeakHour {
				t.Errorf("GetPeakHour() hour = %v, want %v", gotHour, tt.wantPeakHour)
			}
			if gotVal != tt.wantPeakVal {
				t.Errorf("GetPeakHour() val = %v, want %v", gotVal, tt.wantPeakVal)
			}
		})
	}
}

func TestAccountHistoryStats_GetPeakDay(t *testing.T) {
	tests := []struct {
		name        string
		patterns    []WeekdayPattern
		wantPeakDay string
		wantPeakVal float64
	}{
		{
			name:        "Empty",
			patterns:    nil,
			wantPeakDay: "Unknown",
			wantPeakVal: 0,
		},
		{
			name: "SinglePeak",
			patterns: []WeekdayPattern{
				{DayName: "Monday", AvgConsumed: 50.5},
				{DayName: "Tuesday", AvgConsumed: 20.0},
			},
			wantPeakDay: "Monday",
			wantPeakVal: 50.5,
		},
		{
			name: "MultiplePeaks",
			patterns: []WeekdayPattern{
				{DayName: "Friday", AvgConsumed: 10.0},
				{DayName: "Sunday", AvgConsumed: 100.0},
				{DayName: "Saturday", AvgConsumed: 50.0},
			},
			wantPeakDay: "Sunday",
			wantPeakVal: 100.0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &AccountHistoryStats{WeekdayPatterns: tt.patterns}
			gotDay, gotVal := a.GetPeakDay()
			if gotDay != tt.wantPeakDay {
				t.Errorf("GetPeakDay() day = %v, want %v", gotDay, tt.wantPeakDay)
			}
			if gotVal != tt.wantPeakVal {
				t.Errorf("GetPeakDay() val = %v, want %v", gotVal, tt.wantPeakVal)
			}
		})
	}
}
