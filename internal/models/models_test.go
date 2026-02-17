package models

import "testing"

func TestIsValidEntryYearBounds(t *testing.T) {
	current := CurrentEntryYear()

	if !IsValidEntryYear(MinEntryYear) {
		t.Fatalf("expected MinEntryYear=%d to be valid", MinEntryYear)
	}

	if !IsValidEntryYear(current) {
		t.Fatalf("expected current year=%d to be valid", current)
	}

	if IsValidEntryYear(MinEntryYear - 1) {
		t.Fatalf("expected year below MinEntryYear to be invalid")
	}

	if IsValidEntryYear(current + 1) {
		t.Fatalf("expected year above current year to be invalid")
	}
}

func TestAvailableEntryYearsDescending(t *testing.T) {
	years := AvailableEntryYears()
	if len(years) == 0 {
		t.Fatalf("expected available years to be non-empty")
	}

	if years[0] != CurrentEntryYear() {
		t.Fatalf("expected first year to be current year, got %d", years[0])
	}

	last := years[len(years)-1]
	if last != MinEntryYear {
		t.Fatalf("expected last year to be MinEntryYear=%d, got %d", MinEntryYear, last)
	}

	for i := 1; i < len(years); i++ {
		if years[i-1]-years[i] != 1 {
			t.Fatalf("expected consecutive descending years, got %d then %d", years[i-1], years[i])
		}
	}
}
