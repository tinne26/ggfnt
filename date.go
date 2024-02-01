package ggfnt

import "fmt"
import "time"

type Date struct {
	Year uint16
	Month uint8
	Day uint8
}

func CurrentDate() Date {
	year, month, day := time.Now().Date()
	return Date{ Year: uint16(year), Month: uint8(month), Day: uint8(day) }
}

func (self *Date) appendTo(data []byte) []byte {
	return append(appendUint16LE(data, self.Year), self.Month, self.Day)
}

// Doesn't check if the date is in the future, only if it's
// logically valid.
func (self *Date) IsValid() bool {
	if self.Month == 0 && self.Day != 0 { return false }
	if self.Year == 0 && (self.Month != 0 || self.Day != 0) { return false }
	if self.Month > 12 { return false }
	if self.Day > self.monthDays() { return false }
	return true
}

// Returns true only when neither year, month nor day are missing.
func (self *Date) IsComplete() bool {
	return self.Year != 0 && self.Month != 0 && self.Day != 0
}

func (self *Date) HasDay() bool { return self.Day != 0 }
func (self *Date) HasMonth() bool { return self.Month != 0 }
func (self *Date) HasYear() bool { return self.Year != 0 }

func isLeapYear(year uint16) bool {
	if year & 0b11 != 0 { return false }
	if year % 100 != 0 { return true }
	return year % 400 == 0
}

var monthDays = []uint8{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
// Precondition: month is defined and valid.
func (self *Date) monthDays() uint8 {
	if self.Month == 2 && isLeapYear(self.Year) {
		return monthDays[self.Month - 1] + 1
	} else {
		return monthDays[self.Month - 1]
	}
}

// Returns the date in "DAY MonthName YEAR" format (e.g. "1 January 1999").
// If day and/or month are missing, they are not added. If the whole date
// is undefined, "(Unknown Date)" is returned. Invalid dates will be converted
// to string too.
func (self *Date) String() string {
	if self.Day == 0 {
		if self.Month == 0 {
			if self.Year == 0 {
				return "(Unknown Date)"
			}
			return fmt.Sprintf("%04d", self.Year)
		}
		return fmt.Sprintf("%s %04d", self.MonthName(), self.Year)
	}
	return fmt.Sprintf("%d %s %04d", self.Day, self.MonthName(), self.Year)
}

var monthNames = []string{"January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"}
func (self *Date) MonthName() string {
	if self.Month == 0 || self.Month > 12 { return "????" }
	return monthNames[self.Month - 1]
}
