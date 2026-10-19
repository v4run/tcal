package events

import "time"

// day is a month and day of month, without a year.
type day struct {
	m time.Month
	d int
}

// span is an inclusive range of dates. For single-date events end == start;
// Onam runs from Atham to Thiruvonam.
type span struct{ start, end day }

// on returns a single-day span.
func on(m time.Month, d int) span { return span{day{m, d}, day{m, d}} }

// moving holds the per-year dates of events that follow lunar or other
// non-Gregorian calendars. A year with no entry has no occurrence.
//
// Every date below was looked up in a published calendar (October 2026);
// none is computed. A missing year is deliberate: a missing date is
// acceptable, a wrong one is not.
var moving = map[Kind]map[int]span{
	// Makar Sankranti day, New Delhi.
	//
	// Rule: if the Sun enters sidereal Capricorn (the Sankranti moment) before
	// local sunset at New Delhi (about 17:45 IST in mid-January), Sankranti is
	// that day; if after sunset, it is the next day. A year whose moment falls
	// within 15 minutes of sunset would be omitted; none does.
	//
	// Sources:
	//   - AstroSage, https://panchang.astrosage.com/festival/makar-sankranti?date=YYYY
	//     (New Delhi), all years 2026-2040. Dates are derived from its
	//     Sankranti moment / Punya Kaal times, not its header date, which
	//     contradicts its own times in 2030, 2034 and 2038. Moments (IST):
	//     Jan 14 afternoon or morning, before sunset -> Jan 14:
	//       2026 14:49, 2029 09:17, 2030 15:27, 2033 09:55, 2034 16:04,
	//       2037 10:32, 2038 16:41;
	//     Jan 14 after sunset -> Jan 15: 2027 20:59, 2031 21:36, 2035 22:13,
	//       2039 22:50;
	//     early on Jan 15 (Punya Kaal starts at the 07:15 sunrise) -> Jan 15:
	//       2028, 2032, 2036, 2040.
	//   - drikpanchang, https://www.drikpanchang.com/sankranti/makar-sankranti-date-time.html
	//     (New Delhi): 2027 Jan 15.
	//   - drikpanchang Malayalam calendar (Cochin), "Makaram Sankramam",
	//     https://www.drikpanchang.com/malayalam/calendar/malayalam-calendar.html?year=YYYY&geoname-id=1273874:
	//     2027 Jan 15 (Thiruvananthapuram), 2028 Jan 15, 2029 Jan 14.
	//   - dekhopanchang, https://dekhopanchang.com/en/festivals/makar-sankranti/2030:
	//     2030 Jan 14; vedpanchang, https://vedpanchang.com/festivals/makar-sankranti-2034:
	//     2034 Jan 14 (both cited in review).
	//
	// 2030-2040 rest on AstroSage's times (plus the two checks above).
	//
	// Disagreement: the DoPT 2027 restricted-holiday list (F. No. 12/2/2023-JCA,
	// https://www.staffnews.in/2026/07/list-of-restricted-holidays-for-the-year-2027.html)
	// puts Makar Sankranti on Jan 14, 2027. We follow the panchangs (Jan 15,
	// moment after sunset) here even though the gazetted list wins for Holi
	// 2027: Sankranti appears only on the restricted (optional) list, not the
	// closed one, and the same list puts Pongal, the Tamil Nadu observance of
	// the same transit, on Jan 15.
	Sankranti: {
		2026: on(time.January, 14),
		2027: on(time.January, 15),
		2028: on(time.January, 15),
		2029: on(time.January, 14),
		2030: on(time.January, 14),
		2031: on(time.January, 15),
		2032: on(time.January, 15),
		2033: on(time.January, 14),
		2034: on(time.January, 14),
		2035: on(time.January, 15),
		2036: on(time.January, 15),
		2037: on(time.January, 14),
		2038: on(time.January, 14),
		2039: on(time.January, 15),
		2040: on(time.January, 15),
	},

	// Lunar New Year: first day of the first lunar month.
	//
	// Source: Hong Kong Observatory, Gregorian-Lunar Calendar Conversion
	// Tables, https://www.hko.gov.hk/en/gts/time/calendar/text/files/TYYYYe.txt
	// (the row marked "1st Lunar Month"), all years 2026-2040.
	LunarNewYear: {
		2026: on(time.February, 17),
		2027: on(time.February, 6),
		2028: on(time.January, 26),
		2029: on(time.February, 13),
		2030: on(time.February, 3),
		2031: on(time.January, 23),
		2032: on(time.February, 11),
		2033: on(time.January, 31),
		2034: on(time.February, 19),
		2035: on(time.February, 8),
		2036: on(time.January, 28),
		2037: on(time.February, 15),
		2038: on(time.February, 4),
		2039: on(time.January, 24),
		2040: on(time.February, 12),
	},

	// Holi: the day of colours (Rangwali Holi / Dhulandi), New Delhi.
	//
	// Sources:
	//   - drikpanchang, https://www.drikpanchang.com/festivals/holi/festivals-holi-rangwali.html?year=YYYY&geoname-id=1261481
	//     (New Delhi): 2026 Mar 4, 2027 Mar 22, 2028 Mar 11, 2029 Mar 1,
	//     2030 Mar 20, 2031 Mar 9.
	//   - AstroSage, https://panchang.astrosage.com/festival/holi/holi-festival?date=YYYY
	//     (New Delhi), all years 2026-2040.
	//   - DoPT gazetted holidays: 2026 Mar 4 (https://www.staffnews.in/2025/07/list-of-holidays-2026-closed-holidays.html),
	//     2027 Mar 23 (https://www.staffnews.in/2026/07/list-of-holidays-for-the-year-2027-gazetted.html).
	//
	// Disagreement: for 2027 drikpanchang gives Mar 22 but AstroSage and the
	// central-government gazetted list give Mar 23; we use Mar 23 as the date
	// most widely observed in India. 2032-2040 rest on AstroSage, later
	// confirmed in review against qppstudio and vedpanchang.
	Holi: {
		2026: on(time.March, 4),
		2027: on(time.March, 23),
		2028: on(time.March, 11),
		2029: on(time.March, 1),
		2030: on(time.March, 20),
		2031: on(time.March, 9),
		2032: on(time.March, 27),
		2033: on(time.March, 16),
		2034: on(time.March, 5),
		2035: on(time.March, 24),
		2036: on(time.March, 12),
		2037: on(time.March, 2),
		2038: on(time.March, 21),
		2039: on(time.March, 11),
		2040: on(time.March, 29),
	},

	// Vishu as observed in Kerala.
	//
	// Sources:
	//   - Kerala government public holidays 2026 (gazette of 31 Oct 2025, as
	//     reported at https://www.onmanorama.com/news/kerala/2025/10/29/kerala-public-holidays-2026-full-list.html):
	//     2026 Apr 15; drikpanchang Vishu Kani for Cochin
	//     (https://www.drikpanchang.com/festivals/vishu/vishu-kani-date-time.html?geoname-id=1273874)
	//     agrees.
	//   - drikpanchang Malayalam calendar, https://www.drikpanchang.com/malayalam/calendar/malayalam-calendar.html?year=YYYY&geoname-id=1273874
	//     (Cochin; 2027 from the Thiruvananthapuram page, geoname-id=1254163):
	//     2027 Apr 15, 2028 Apr 14, 2029 Apr 14.
	//   - drikpanchang Vishu Kani for Cochin,
	//     https://www.drikpanchang.com/festivals/vishu/vishu-kani-date-time.html?year=YYYY&geoname-id=1273874:
	//     2030 Apr 15, 2031 Apr 15.
	//
	// Disagreement: the DoPT 2027 restricted list has "Vaisakhi Vishu" on
	// Apr 14, 2027; Kerala observes Apr 15, which we use.
	//
	// 2032-2040 are omitted: drikpanchang (the only Kerala-specific source
	// found) rate-limited further lookups, and Vishu flips between Apr 14 and
	// Apr 15 by the Kerala sunrise rule, so it cannot be guessed.
	Vishu: {
		2026: on(time.April, 15),
		2027: on(time.April, 15),
		2028: on(time.April, 14),
		2029: on(time.April, 14),
		2030: on(time.April, 15),
		2031: on(time.April, 15),
	},

	// Eid al-Fitr (1 Shawwal), Umm al-Qura calendar of Saudi Arabia, which is
	// computed astronomically and published years ahead.
	//
	// Sources:
	//   - al-habib.info Umm al-Qura calendars,
	//     https://www.al-habib.info/islamic-calendar/ummulqura/islamic-calendar-ummulqura-YYYY-ce.htm,
	//     all years 2026-2040.
	//   - qppstudio (Umm al-Qura), https://www.qppstudio.net/global-holidays-observances/eid-al-fitr-end-of-ramadan.htm,
	//     2026-2036: agrees in every year.
	//   - Wikipedia, https://en.wikipedia.org/wiki/Eid_al-Fitr ("Umm al-Qura
	//     predicted"), 2026-2030: agrees except 2030 (Feb 3); al-habib,
	//     qppstudio and ICU's islamic-umalqura calendar all give Feb 4.
	//
	// Disagreements:
	//   - Countries that rely on moon sighting often celebrate a day later:
	//     timeanddate.com expects Mar 10, 2027 in most countries, and India's
	//     central government lists Mar 21, 2026 and Mar 10, 2027. Kerala's
	//     2026 list matches Umm al-Qura (Mar 20). We keep the astronomical date.
	//   - 2033 has two Eids (Jan 2 and Dec 23). The table holds one date per
	//     year, so we keep Dec 23, on which all sources agree (ICU gives Jan 3
	//     for the January one).
	//   - 2037 is omitted: al-habib gives Nov 8, ICU and miqatona.com Nov 9,
	//     and timeanddate's Saudi listing Nov 10.
	Eid: {
		2026: on(time.March, 20),
		2027: on(time.March, 9),
		2028: on(time.February, 26),
		2029: on(time.February, 14),
		2030: on(time.February, 4),
		2031: on(time.January, 24),
		2032: on(time.January, 14),
		2033: on(time.December, 23),
		2034: on(time.December, 12),
		2035: on(time.December, 1),
		2036: on(time.November, 19),
		2038: on(time.October, 29),
		2039: on(time.October, 19),
		2040: on(time.October, 7),
	},

	// Onam: Atham (Atthachamayam, day 1) to Thiruvonam, Malayalam calendar.
	//
	// Source: drikpanchang Onam calendar for Cochin,
	// https://www.drikpanchang.com/festivals/onam/onam-calendar.html?year=YYYY&geoname-id=1273874.
	// Thiruvonam cross-checked against the Kerala 2026 holiday list (Aug 26),
	// the DoPT 2027 restricted list (Sep 12) and drikpanchang's Malayalam
	// calendar (2028 Sep 1, 2029 Aug 22). AstroSage's Thiruvonam is computed
	// for New Delhi and differs in 2027 (Sep 13), so it was not used.
	//
	// 2031-2040 are omitted: drikpanchang rate-limited further lookups and no
	// other source found gives Atham for Kerala.
	Onam: {
		2026: {day{time.August, 16}, day{time.August, 26}},
		2027: {day{time.September, 3}, day{time.September, 12}},
		2028: {day{time.August, 23}, day{time.September, 1}},
		2029: {day{time.August, 13}, day{time.August, 22}},
		2030: {day{time.August, 31}, day{time.September, 9}},
	},

	// Diwali: the Lakshmi Puja day, New Delhi.
	//
	// Sources:
	//   - drikpanchang, https://www.drikpanchang.com/festivals/lakshmipuja/festivals-lakshmipuja-timings.html?year=2026
	//     (New Delhi), "Lakshmi Puja in Recent Years" table, 2026-2033.
	//   - AstroSage, https://panchang.astrosage.com/festival/diwali?date=YYYY
	//     (New Delhi), all years 2026-2040; agrees with drikpanchang 2026-2033.
	//   - DoPT gazetted holidays: 2026 Nov 8, 2027 Oct 29 (agree).
	//
	// 2034-2040 rest on AstroSage, later confirmed in review against
	// qppstudio, vedpanchang and AstroSage's Amavasya timings.
	//
	// 2037 is omitted: AstroSage gives Nov 7, but its Amavasya runs Nov 6 20:54
	// to Nov 7 17:34, leaving almost no pradosh on either day, and vedpanchang
	// and drikpanchang (Columbus) give Nov 6; no India source settles it.
	Diwali: {
		2026: on(time.November, 8),
		2027: on(time.October, 29),
		2028: on(time.October, 17),
		2029: on(time.November, 5),
		2030: on(time.October, 26),
		2031: on(time.November, 14),
		2032: on(time.November, 2),
		2033: on(time.October, 22),
		2034: on(time.November, 10),
		2035: on(time.October, 30),
		2036: on(time.October, 18),
		2038: on(time.October, 27),
		2039: on(time.November, 15),
		2040: on(time.November, 4),
	},
}
