// Web interface generator
//
// Copyright (c) 2021, 2022  Philip Kaludercic
//
// This file is part of go-kgp.
//
// go-kgp is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License,
// version 3, as published by the Free Software Foundation.
//
// go-kgp is distributed in the hope that it will be useful, but
// WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU
// Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public
// License, version 3, along with go-kgp. If not, see
// <http://www.gnu.org/licenses/>

package web

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"time"

	"go-kgp"
)

const PER_PAGE = 50

//go:embed static
var static embed.FS

//go:embed *.tmpl
var html embed.FS

var (
	// Template manager
	tmpl *template.Template

	// Custom template functions
	funcs = template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
		"dec": func(i int) int {
			return i - 1
		},
		"timefmt": func(t time.Time) string {
			s := time.Since(t).Round(time.Second)
			switch {
			case s < time.Second*5:
				return "now"
			case s < time.Minute:
				return fmt.Sprintf("%.1gs ago", s.Seconds())
			case s < 10*time.Minute:
				return fmt.Sprintf("%.3gm ago", s.Minutes())
			default:
				return t.Format(time.Stamp)
			}
		},
		"result": func(a *kgp.User, g kgp.Game) template.HTML {
			var msg string
			switch g.Outcome {
			case kgp.WIN:
				msg = "South won"
			case kgp.DRAW:
				msg = "Draw"
			case kgp.LOSS:
				msg = "North won"
			case kgp.RESIGN:
				msg = "Resignation"
			case kgp.ONGOING:
				return "Ongoing"
			default:
				panic("Unknown outcome")
			}
			if a == nil {
				return template.HTML(msg)
			}

			switch {
			case g.North != nil && g.North.User() != nil && a.Id == g.North.User().Id:
				// Reminder: the outcome is stored
				// from the perspective of the
				// southern player
				switch g.Outcome {
				case kgp.LOSS:
					msg = `<span class="won">` + msg + `</span>`
				case kgp.WIN:
					msg = `<span class="lost">` + msg + `</span>`
				case kgp.DRAW:
					msg = `<span class="draw">` + msg + `</span>`
				case kgp.RESIGN:
					msg = `<span class="resign">` + msg + `</span>`
				}
			case g.South != nil && g.South.User() != nil && a.Id == g.South.User().Id:
				switch g.Outcome {
				case kgp.LOSS:
					msg = `<span class="lost">` + msg + `</span>`
				case kgp.WIN:
					msg = `<span class="won">` + msg + `</span>`
				case kgp.DRAW:
					msg = `<span class="draw">` + msg + `</span>`
				case kgp.RESIGN:
					msg = `<span class="resign">` + msg + `</span>`
				}
			}

			return template.HTML(msg)
		},
		"now": func() string {
			return time.Now().Format(time.RFC3339)
		},
		"hasMore": func(i int) bool {
			return i%PER_PAGE != 0
		},
		"are": func(n uint64) string {
			if n == 1 {
				return "is"
			}
			return "are"
		},
		"same": func(a *kgp.User, b kgp.Agent) bool {
			return (a != nil && b != nil && a.Id == b.User().Id)
		},
		"user": func(a kgp.Agent) *kgp.User {
			return a.User()
		},
		"describe": func(outcome kgp.Outcome) string {
			switch outcome {
			case kgp.ONGOING:
				return "The match is still ongoing."
			case kgp.WIN:
				return "The southern player won the match."
			case kgp.DRAW:
				return "The match resulted in a draw."
			case kgp.LOSS:
				return "The northern player won the match."
			case kgp.RESIGN:
				return "The match was prematurely terminated."
			}
			panic("Illegal game state")
		},
		"board": func(b *kgp.Board) string {
			size, init := b.Type()
			return fmt.Sprintf("(%d, %d)", size, init)
		},
		"draw": func(m *kgp.Move, g *kgp.Game) template.HTML {
			var (
				B       bytes.Buffer
				b       = m.State
				size, _ = b.Type()
				u       = 50.0
				w       = (u*2 + u*float64(size))
			)

			circle := func(x, y float64, n uint, hl bool) {
				// https://developer.mozilla.org/en-US/docs/Web/SVG/Element/circle
				if x < 0 {
					x = float64(size) - x
				}
				color := "sienna"
				if hl {
					color = "seagreen"
				}
				fmt.Fprintf(&B, `<circle fill="%s" cx="%g" cy="%g" r="%g" />`,
					color, u*x+u/2, u*y+u/2, u*0.8/2)
				// https://developer.mozilla.org/en-US/docs/Web/SVG/Element/text
				d := u * .4
				if n >= 10 {
					d = u * 0.3
				} else if n >= 100 {
					d = u * 0.2
				}
				fmt.Fprintf(&B, `<text x="%g" y="%g">%d</text>`,
					u*x+d, 0.4*u+u*y+u*.2, n)
			}

			fmt.Fprintf(&B, `<svg width="%g" height="%g">`, w, 2*u)

			// https://developer.mozilla.org/en-US/docs/Web/SVG/Element/rect
			fmt.Fprintf(&B, `<rect x="0" y="0" rx="10" ry="10" width="%g" height="%g" fill="burlywood" />`,
				w, 2*u)
			for i := int(size - 1); i >= 0; i-- {
				n := b.Pit(kgp.North, uint(i))
				hl := g.North == m.Agent && m.Choice == uint(i)
				circle(float64(1+i), 0, n, hl)
			}
			for i := int(0); i < int(size); i++ {
				s := b.Pit(kgp.South, uint(i))
				hl := g.South == m.Agent && m.Choice == uint(i)
				circle(float64(1+i), 1, s, hl)
			}
			circle(0.1, 0.5, b.Store(kgp.North), false)
			circle(-0.9, 0.5, b.Store(kgp.South), false)

			fmt.Fprintf(&B, `</svg>`)

			return template.HTML(B.String())
		},
	}
)