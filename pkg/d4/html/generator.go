package html

import (
	"fmt"
	"github.com/Dakota628/d4parse/pkg/d4"
	"html"
	"reflect"
	"strings"
	"unicode"
)

type Generator struct {
	sb         strings.Builder
	tocEntries d4.TocEntries
	gbData     *d4.GbData
}

func NewGenerator(toc d4.Toc, gbData *d4.GbData) *Generator {
	return &Generator{
		tocEntries: toc.Entries,
		gbData:     gbData,
	}
}

func (g *Generator) genericType(rtStr string) string {
	i := strings.IndexByte(rtStr, '[')
	if i <= 0 {
		return rtStr
	}
	return rtStr[:i]
}

func (g *Generator) genericField(rv reflect.Value, field string) any {
	return rv.Elem().FieldByName(field).Interface()
}

func (g *Generator) prettyTypeName(t any) string {
	typeName := fmt.Sprintf("%T", t)
	typeName = strings.Replace(typeName, "*github.com/Dakota628/d4parse/pkg/d4.", "", -1)
	typeName = strings.Replace(typeName, "*d4.", "", -1)
	typeName = strings.Replace(typeName, "d4.", "", -1)
	return typeName
}

func (g *Generator) prettyFieldName(fieldName string) string {
	r := []rune(fieldName)
	r[0] = unicode.ToLower(r[0])
	return string(r)
}

func (g *Generator) writeFmt(format string, a ...any) {
	g.sb.WriteString(fmt.Sprintf(format, a...)) // TODO: utilize Fprintf
}

func (g *Generator) walkCallback(k string, x d4.Object, next d4.WalkNext, d ...any) {
	// Determine how to display the value
	switch {
	// It's an element of an iterable
	case d4.IsIndex(k):
		g.sb.WriteString("<li>")
		defer g.sb.WriteString("</li>")

	// It's a root element
	case k == "":
		break

	// It's a field value
	default:
		// Generate extra field attributes
		var extraFieldAttrs string
		if k == "DwUID" { // Add ids for UIDs for fragment targets
			if id, ok := x.(*d4.DT_UINT); ok {
				extraFieldAttrs = fmt.Sprintf(`id="uid%v"`, id)
			}
		}

		// Write the field HTML
		g.writeFmt(
			`<div %sclass="f"><div class="fk"><span class="fn">%s</span><span class="ft">%s</span></div>`,
			extraFieldAttrs,
			g.prettyFieldName(k),
			g.prettyTypeName(x),
		)
		g.sb.WriteString(`<div class="fv">`)
		defer g.sb.WriteString(`</div></div>`)
	}

	// Fast path
	switch t := x.(type) {
	case *d4.SnoMeta:
		group, name := g.tocEntries.GetName(t.Id.Value)

		var prefix string
		if group == d4.SnoGroupStringList {
			prefix = "enUS_Text"
		} else {
			prefix = "base"
		}

		g.sb.WriteString(`<div class="t snoMeta"><div class="tn">SNO Info</div>`)
		g.sb.WriteString(`<div class="f"><div class="fk"><div class="fn">Group</div></div>`)
		g.writeFmt(`<div class="fv"><p>%s</p></div></div>`, group)
		g.sb.WriteString(`<div class="f"><div class="fk"><div class="fn">ID</div></div>`)
		g.writeFmt(`<div class="fv"><p>%d</p></div></div>`, t.Id.Value)
		g.sb.WriteString(`<div class="f"><div class="fk"><div class="fn">Name</div></div>`)
		g.writeFmt(`<div class="fv"><p>%s</p></div></div>`, name)
		g.sb.WriteString(`<div class="f"><div class="fk"><div class="fn">File</div></div>`)
		g.writeFmt(`<div class="fv"><p>%s/meta/%s/%s%s</p></div></div>`, prefix, group, name, group.Ext())
		g.sb.WriteString("</div>")
		return
	case *d4.DT_NULL:
		return
	case *d4.DT_BYTE:
		g.writeFmt("<p>0x%x</p>", t.Value)
		return
	case *d4.DT_WORD:
		g.writeFmt("<p>0x%x</p>", t.Value)
		return
	case *d4.DT_ENUM:
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_INT:
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_FLOAT:
		g.writeFmt("<p>%f</p>", t.Value)
		return
	case *d4.DT_SNO:
		if t.Id > 0 {
			group, name := g.tocEntries.GetName(t.Id)
			g.writeFmt(`<p><a class="snoRef" href="../sno/%d.html">[%s] %s</a></p>`, t.Id, group, name)
		}
		return
	case *d4.DT_SNO_NAME:
		if t.Id > 0 {
			group, name := g.tocEntries.GetName(t.Id, d4.SnoGroup(t.Group))
			g.writeFmt(`<p><a class="snoRef" href="../sno/%d.html">[%s] %s</a></p>`, t.Id, group, name)
		}
		return
	case *d4.DT_GBID:
		if t.Value == 0 || t.Value == 0xFFFFFFFF || t.Group == 0 || t.Group == -1 {
			return
		}

		if gbInfoIfc, ok := g.gbData.Load(*t); ok {
			if gbInfo, ok := gbInfoIfc.(d4.GbInfo); ok {
				_, gbIdSnoName := g.tocEntries.GetName(gbInfo.SnoId)
				g.writeFmt(
					`<p><a class="gbidRef" href="../sno/%d.html#gbid%d">[%s] %s</a><p>`,
					gbInfo.SnoId, t.Value, gbIdSnoName, gbInfo.Name,
				)
				return
			}
		}

		g.writeFmt(`<p>%d <i>(unknown GBID)</i></p>`, t.Value)
		return
	case *d4.DT_STARTLOC_NAME:
		// TODO: can it be enriched?
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_UINT:
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_ACD_NETWORK_NAME:
		// TODO: can it be enriched?
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_SHARED_SERVER_DATA_ID:
		// TODO: can it be enriched?
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_INT64:
		g.writeFmt("<p>%d</p>", t.Value)
		return
	case *d4.DT_STRING_FORMULA:
		g.sb.WriteString(`<code class="formula">`)
		g.sb.WriteString(html.EscapeString(t.Value))
		g.sb.WriteString("</code>")
		return
	case *d4.DT_CHARARRAY:
		g.sb.WriteString("<pre>")
		g.sb.WriteString(html.EscapeString(string(t.Value)))
		g.sb.WriteString("</pre>")
		return
	case *d4.DT_RGBACOLOR:
		g.writeFmt("<p>#%x%x%x%x</p>", t.B, t.G, t.B, t.A) // TODO: sample color box next to text
		return
	case *d4.DT_RGBACOLORVALUE:
		g.writeFmt("<p>rgba(%f, %f, %f, %f)</p>", t.B, t.G, t.B, t.A) // TODO: sample color box to text
		return
	case *d4.DT_BCVEC2I:
		g.writeFmt("<p>(%f, %f)</p>", t.X, t.Y)
		return
	case *d4.DT_VECTOR2D:
		g.writeFmt("<p>(%f, %f)</p>", t.X, t.Y)
		return
	case *d4.DT_VECTOR3D:
		g.writeFmt("<p>(%f, %f, %f)</p>", t.X, t.Y, t.Z)
		return
	case *d4.DT_VECTOR4D:
		g.writeFmt("<p>(%f, %f, %f, %f)</p>", t.X, t.Y, t.Z, t.W)
		return
	case nil:
		g.sb.WriteString("<p><i>note: could not obtain element</i></p>") // This should only happen on invalid polymorphic arrays
		return
	}

	// Slow path (reflection)
	rt := reflect.TypeOf(x)
	baseTypeName := g.genericType(rt.String())

	switch baseTypeName {
	case "*d4.DT_OPTIONAL":
		next()
		return
	case "*d4.DT_RANGE":
		g.sb.WriteString(`<div class="t">`)
		next()
		g.sb.WriteString("</div>")
		return
	case "*d4.DT_FIXEDARRAY", "*d4.DT_VARIABLEARRAY", "*d4.DT_POLYMORPHIC_VARIABLEARRAY":
		if maybeExt, ok := x.(d4.MaybeExternal); ok && maybeExt.IsExternal() {
			g.sb.WriteString("<p><i>note: external data is not supported</i></p>") // TODO
			return
		}

		g.sb.WriteString(`<ul class="arr">`)
		next()
		g.sb.WriteString("</ul>")
		return
	case "*d4.DT_TAGMAP":
		g.sb.WriteString("<p><i>note: tag map parsing is not supported</i></p>") // TODO
		return
	case "*d4.DT_CSTRING":
		g.sb.WriteString("<pre>")
		g.sb.WriteString(html.EscapeString(fmt.Sprintf("%s", x)))
		g.sb.WriteString("</pre>")
		return
	default: // Non-basic types
		// Write type header (with specific attributes for linking)
		g.sb.WriteString("<div ")
		switch t := x.(type) {
		case *d4.GBIDHeader:
			g.writeFmt(`id="gbid%d" `, d4.GbidHash(string(t.SzName.Value)))
		}
		g.sb.WriteString(`class="t">`)

		// Write type
		g.writeFmt(`<div class="tn">%s</div>`, g.prettyTypeName(x))
		next()
		g.sb.WriteString("</div>")

		return
	}
}

func (g *Generator) Add(x d4.Object) {
	g.walkCallback("", x, func(d ...any) {})
	if w, ok := x.(d4.Walkable); ok {
		w.Walk(g.walkCallback)
	}
}

func (g *Generator) String() string {
	return fmt.Sprintf(
		`<html lang="en"><head><script data-main="../js/sno" src="https://cdnjs.cloudflare.com/ajax/libs/require.js/2.3.6/require.min.js"></script><link rel="stylesheet" href="../css/sno.css"></head><body>%s</body></html>`,
		g.sb.String(),
	)
}
