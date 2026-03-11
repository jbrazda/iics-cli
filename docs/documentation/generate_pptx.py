"""
Generate iics-cli presentation as a PowerPoint file.
Run: python3 generate_pptx.py
"""

from pptx import Presentation
from pptx.util import Inches, Pt, Emu
from pptx.dml.color import RGBColor
from pptx.enum.text import PP_ALIGN
from pptx.util import Inches, Pt
import copy

# ── Colour palette ───────────────────────────────────────────────────────────
DARK_BLUE   = RGBColor(0x00, 0x33, 0x66)   # headings / title bg
MID_BLUE    = RGBColor(0x00, 0x66, 0xCC)   # accents, table header
ORANGE      = RGBColor(0xFF, 0x66, 0x00)   # highlight colour
WHITE       = RGBColor(0xFF, 0xFF, 0xFF)
LIGHT_GRAY  = RGBColor(0xF2, 0xF2, 0xF2)
DARK_GRAY   = RGBColor(0x33, 0x33, 0x33)
MID_GRAY    = RGBColor(0x88, 0x88, 0x88)
GREEN       = RGBColor(0x00, 0x99, 0x44)
RED         = RGBColor(0xCC, 0x00, 0x00)
CODE_BG     = RGBColor(0x1E, 0x1E, 0x2E)   # dark code background
CODE_FG     = RGBColor(0xA6, 0xE3, 0xA1)   # green code text

SLIDE_W = Inches(13.33)
SLIDE_H = Inches(7.5)

prs = Presentation()
prs.slide_width  = SLIDE_W
prs.slide_height = SLIDE_H

blank_layout = prs.slide_layouts[6]   # completely blank


# ── helpers ──────────────────────────────────────────────────────────────────

def add_rect(slide, x, y, w, h, fill_rgb, alpha=None):
    shape = slide.shapes.add_shape(1, x, y, w, h)   # MSO_SHAPE_TYPE.RECTANGLE = 1
    shape.line.fill.background()
    shape.fill.solid()
    shape.fill.fore_color.rgb = fill_rgb
    return shape

def add_text_box(slide, text, x, y, w, h,
                 font_size=18, bold=False, italic=False,
                 color=DARK_GRAY, align=PP_ALIGN.LEFT,
                 word_wrap=True, font_name="Calibri"):
    txBox = slide.shapes.add_textbox(x, y, w, h)
    txBox.word_wrap = word_wrap
    tf = txBox.text_frame
    tf.word_wrap = word_wrap
    p = tf.paragraphs[0]
    p.alignment = align
    run = p.add_run()
    run.text = text
    run.font.size = Pt(font_size)
    run.font.bold = bold
    run.font.italic = italic
    run.font.color.rgb = color
    run.font.name = font_name
    return txBox

def add_multiline_text(slide, lines, x, y, w, h,
                       font_size=16, color=DARK_GRAY,
                       font_name="Calibri", line_spacing=None):
    """lines: list of (text, bold, italic, color_override)"""
    txBox = slide.shapes.add_textbox(x, y, w, h)
    txBox.word_wrap = True
    tf = txBox.text_frame
    tf.word_wrap = True
    first = True
    for item in lines:
        if isinstance(item, str):
            text, bold, italic, col = item, False, False, color
        else:
            text = item[0]
            bold = item[1] if len(item) > 1 else False
            italic = item[2] if len(item) > 2 else False
            col = item[3] if len(item) > 3 else color
        if first:
            p = tf.paragraphs[0]
            first = False
        else:
            p = tf.add_paragraph()
        if line_spacing:
            p.space_before = Pt(line_spacing)
        run = p.add_run()
        run.text = text
        run.font.size = Pt(font_size)
        run.font.bold = bold
        run.font.italic = italic
        run.font.color.rgb = col
        run.font.name = font_name
    return txBox

def slide_header(slide, title, subtitle=None):
    """Blue header bar across the top of the slide."""
    add_rect(slide, 0, 0, SLIDE_W, Inches(1.1), DARK_BLUE)
    add_text_box(slide, title,
                 Inches(0.3), Inches(0.1), Inches(12.5), Inches(0.7),
                 font_size=28, bold=True, color=WHITE,
                 align=PP_ALIGN.LEFT)
    if subtitle:
        add_text_box(slide, subtitle,
                     Inches(0.3), Inches(0.75), Inches(12.5), Inches(0.4),
                     font_size=14, color=RGBColor(0xBB, 0xCC, 0xFF),
                     align=PP_ALIGN.LEFT)

def footer(slide, text="iics-cli  |  Informatica IICS Command Line Interface"):
    add_rect(slide, 0, Inches(7.2), SLIDE_W, Inches(0.3), DARK_BLUE)
    add_text_box(slide, text,
                 Inches(0.3), Inches(7.18), Inches(12.7), Inches(0.3),
                 font_size=9, color=RGBColor(0xAA, 0xBB, 0xCC),
                 align=PP_ALIGN.LEFT)

def code_block(slide, code, x, y, w, h, font_size=11):
    add_rect(slide, x, y, w, h, CODE_BG)
    add_text_box(slide, code, x + Inches(0.15), y + Inches(0.1),
                 w - Inches(0.3), h - Inches(0.2),
                 font_size=font_size, color=CODE_FG,
                 font_name="Courier New")

def bullet_list(slide, items, x, y, w, h, font_size=16,
                indent=Inches(0.3), color=DARK_GRAY,
                bullet="•"):
    txBox = slide.shapes.add_textbox(x, y, w, h)
    txBox.word_wrap = True
    tf = txBox.text_frame
    tf.word_wrap = True
    first = True
    for item in items:
        if isinstance(item, str):
            text, bold, col = item, False, color
        else:
            text = item[0]
            bold = item[1] if len(item) > 1 else False
            col = item[2] if len(item) > 2 else color
        if first:
            p = tf.paragraphs[0]
            first = False
        else:
            p = tf.add_paragraph()
        run = p.add_run()
        run.text = f"{bullet}  {text}"
        run.font.size = Pt(font_size)
        run.font.bold = bold
        run.font.color.rgb = col
        run.font.name = "Calibri"
    return txBox


def add_table(slide, headers, rows, x, y, w, h,
              header_color=MID_BLUE, alt_row=True):
    num_cols = len(headers)
    num_rows = len(rows) + 1  # +1 for header
    tbl = slide.shapes.add_table(num_rows, num_cols, x, y, w, h).table

    col_w = w // num_cols
    for i in range(num_cols):
        tbl.columns[i].width = col_w

    # Header row
    for ci, hdr in enumerate(headers):
        cell = tbl.cell(0, ci)
        cell.fill.solid()
        cell.fill.fore_color.rgb = header_color
        p = cell.text_frame.paragraphs[0]
        p.alignment = PP_ALIGN.CENTER
        run = p.add_run()
        run.text = hdr
        run.font.bold = True
        run.font.size = Pt(13)
        run.font.color.rgb = WHITE
        run.font.name = "Calibri"

    # Data rows
    for ri, row in enumerate(rows):
        bg = LIGHT_GRAY if (alt_row and ri % 2 == 0) else WHITE
        for ci, val in enumerate(row):
            cell = tbl.cell(ri + 1, ci)
            cell.fill.solid()
            cell.fill.fore_color.rgb = bg
            p = cell.text_frame.paragraphs[0]
            p.alignment = PP_ALIGN.CENTER if ci > 0 else PP_ALIGN.LEFT
            run = p.add_run()
            # colour Yes/No
            if val == "Yes":
                run.font.color.rgb = GREEN
                run.font.bold = True
            elif val == "No":
                run.font.color.rgb = RED
            else:
                run.font.color.rgb = DARK_GRAY
            run.text = val
            run.font.size = Pt(12)
            run.font.name = "Calibri"
    return tbl


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 1 – Title
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
add_rect(s, 0, 0, SLIDE_W, SLIDE_H, DARK_BLUE)
add_rect(s, 0, Inches(3.5), SLIDE_W, Inches(0.06), ORANGE)

add_text_box(s, "iics-cli",
             Inches(1), Inches(1.2), Inches(11), Inches(1.4),
             font_size=72, bold=True, color=WHITE, align=PP_ALIGN.CENTER)
add_text_box(s, "A Modern Command-Line Interface for Informatica IICS",
             Inches(1), Inches(2.6), Inches(11), Inches(0.8),
             font_size=24, color=RGBColor(0xBB, 0xCC, 0xFF),
             align=PP_ALIGN.CENTER)
add_text_box(s, "REST API v3  |  Go Binary  |  POSIX Compliant  |  Actively Maintained",
             Inches(1), Inches(3.7), Inches(11), Inches(0.6),
             font_size=18, color=RGBColor(0xFF, 0xAA, 0x44),
             align=PP_ALIGN.CENTER)
add_text_box(s, "Replacing the abandoned Informatica Asset Management CLI V2",
             Inches(1), Inches(4.5), Inches(11), Inches(0.5),
             font_size=16, italic=True, color=MID_GRAY, align=PP_ALIGN.CENTER)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 2 – Agenda
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Agenda")
footer(s)

items_l = [
    "1.  The Problem – Limitations of the Official Informatica CLI",
    "2.  POSIX Exit Codes – Reliable CI/CD Integration",
    "3.  Machine-Readable Output – JSON, CSV, Tables",
    "4.  Secure Credential Management",
    "5.  Named Environment Profiles",
]
items_r = [
    "6.  Session Caching",
    "7.  Broad API Coverage – 20+ Resources",
    "8.  Full Region Support",
    "9.  Single Binary – No JVM Required",
    "10. Feature Comparison Summary",
]

bullet_list(s, items_l, Inches(0.5), Inches(1.3), Inches(6), Inches(5.5),
            font_size=17, bullet="")
bullet_list(s, items_r, Inches(6.8), Inches(1.3), Inches(6), Inches(5.5),
            font_size=17, bullet="")


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 3 – The Problem
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "The Problem", "Informatica Asset Management CLI V2 – what it cannot do")
footer(s)

problems = [
    ("No longer maintained – last update years ago, no bug fixes", True, RED),
    ("Covers only 5 operations: export, import, publish, list, version", False, DARK_GRAY),
    ("Always exits with code 0 – CI/CD pipelines cannot detect failures", False, DARK_GRAY),
    ("No structured output – cannot pipe to jq, grep, or other UNIX tools", False, DARK_GRAY),
    ("Credentials passed as command-line flags (-u user -p pass) – visible in ps, logs, history", False, DARK_GRAY),
    ("No environment profiles – must retype credentials for every org/env switch", False, DARK_GRAY),
    ("Authenticates on every invocation – no session reuse", False, DARK_GRAY),
    ("Only 3 hardcoded regions: us / eu / ap – many IICS pods not covered", False, DARK_GRAY),
    ("Java-based – requires JVM on every machine / Docker image", False, DARK_GRAY),
    ("Flat command model – no subcommand hierarchy, poor discoverability", False, DARK_GRAY),
]
bullet_list(s, problems, Inches(0.5), Inches(1.25), Inches(12.5), Inches(5.8),
            font_size=15)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 4 – POSIX Exit Codes
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "POSIX Exit Codes", "Reliable failure detection in scripts and CI/CD pipelines")
footer(s)

add_text_box(s, "Official CLI – exit code is always 0",
             Inches(0.5), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=RED)
code_block(s, (
    "# Export fails – but exit code is still 0\n"
    "iics export -u user@co.com -p secret -r us \\\n"
    "  -a \"MyProject/BadAsset.Process\"\n"
    "echo $?   # → 0  (pipeline keeps running!)"
), Inches(0.5), Inches(1.65), Inches(5.8), Inches(1.6), font_size=11)

add_text_box(s, "iics-cli – three distinct exit codes",
             Inches(7.0), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=GREEN)
code_block(s, (
    "iics export create --name nightly --project Prod\n"
    "echo $?   # → 1  (API / runtime error)\n\n"
    "iics export create --bad-flag\n"
    "echo $?   # → 2  (usage / flag error)"
), Inches(7.0), Inches(1.65), Inches(5.8), Inches(1.6), font_size=11)

add_table(s,
    ["Exit Code", "Meaning", "Use Case"],
    [
        ["0", "Success", "All pipeline steps continue"],
        ["1", "Runtime / API error", "API call failed, network error"],
        ["2", "Usage error", "Unknown flag, missing argument"],
    ],
    Inches(2.0), Inches(3.5), Inches(9.0), Inches(1.7))

add_text_box(s,
    "Works correctly with set -e scripts, make targets, GitHub Actions, and Jenkins.",
    Inches(0.5), Inches(5.4), Inches(12.3), Inches(0.5),
    font_size=14, italic=True, color=MID_BLUE)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 5 – Output Formats
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Machine-Readable Output", "JSON, CSV, and Table formats – pipe into any UNIX tool")
footer(s)

add_text_box(s, "Every command supports --output / -o  table | json | csv",
             Inches(0.5), Inches(1.2), Inches(12.5), Inches(0.4),
             font_size=15, bold=True, color=DARK_BLUE)

code_block(s, (
    "# Default table – human readable\n"
    "iics user list\n\n"
    "# JSON – pipe into jq\n"
    "iics user list -o json | jq -r '.[].userName'\n\n"
    "# CSV – import into Excel / Google Sheets\n"
    "iics connection list -o csv > connections.csv\n\n"
    "# Find enabled schedules and extract names\n"
    "iics schedule list -o json \\\n"
    "  | jq -r '.[] | select(.status==\"ENABLED\") | .name'\n\n"
    "# Count objects in a project\n"
    "iics objects list --project Prod -o json | jq length"
), Inches(0.5), Inches(1.7), Inches(12.3), Inches(3.9), font_size=11)

add_text_box(s,
    "stdout is always clean data.  All errors and diagnostics go to stderr.",
    Inches(0.5), Inches(5.75), Inches(12.3), Inches(0.4),
    font_size=14, italic=True, color=MID_BLUE)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 6 – Secure Credentials
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Secure Credential Management", "Credentials never appear on the command line")
footer(s)

add_text_box(s, "Official CLI – credentials on every command line",
             Inches(0.5), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=RED)
code_block(s, (
    "# Visible in ps aux, shell history, CI logs\n"
    "iics export \\\n"
    "  -u admin@company.com \\\n"
    "  -p MyP@ssword123 \\\n"
    "  -r us \\\n"
    "  -a \"Prod/CriticalFlow.Process\""
), Inches(0.5), Inches(1.65), Inches(5.8), Inches(2.0), font_size=11)

add_text_box(s, "iics-cli – credentials in config file + env var",
             Inches(7.0), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=GREEN)
code_block(s, (
    "# ~/.iics/config.yaml  (chmod 600)\n"
    "defaultProfile: prod\n"
    "profiles:\n"
    "  prod:\n"
    "    region: US\n"
    "    username: admin@company.com\n\n"
    "# Password via env var – never in a file\n"
    "export IICS_PASSWORD='MyP@ssword123'\n"
    "iics export create --name nightly"
), Inches(7.0), Inches(1.65), Inches(5.8), Inches(2.0), font_size=11)

bullet_list(s, [
    "Config file protected with OS file permissions (chmod 600)",
    "Password sourced from IICS_PASSWORD env var or secret manager",
    "No credentials in process list, shell history, or build logs",
    "Compatible with HashiCorp Vault, AWS Secrets Manager, GitHub Secrets",
], Inches(0.5), Inches(3.9), Inches(12.3), Inches(2.4), font_size=15)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 7 – Named Profiles
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Named Environment Profiles", "Switch between orgs and environments with a single flag")
footer(s)

code_block(s, (
    "# ~/.iics/config.yaml\n"
    "defaultProfile: dev\n"
    "profiles:\n"
    "  dev:\n"
    "    region: US\n"
    "    username: dev-user@company.com\n"
    "  staging:\n"
    "    region: US\n"
    "    username: staging-admin@company.com\n"
    "  prod:\n"
    "    region: US\n"
    "    username: prod-admin@company.com\n"
    "  emea:\n"
    "    region: EMEA\n"
    "    username: emea-admin@company.com"
), Inches(0.5), Inches(1.3), Inches(5.8), Inches(4.5), font_size=11)

add_text_box(s, "Switching environments",
             Inches(7.0), Inches(1.3), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=DARK_BLUE)
code_block(s, (
    "# Use the --profile flag\n"
    "iics user list --profile dev\n"
    "iics user list --profile prod\n\n"
    "# Or the IICS_PROFILE env var\n"
    "IICS_PROFILE=emea iics export create \\\n"
    "  --name weekly-backup\n\n"
    "# Perfect for deployment scripts\n"
    "for env in dev staging prod; do\n"
    "  iics export create --profile $env \\\n"
    "    --name \"snapshot-$env\"\n"
    "done"
), Inches(7.0), Inches(1.75), Inches(5.8), Inches(3.0), font_size=11)

bullet_list(s, [
    "Separate credentials per environment – no copy-paste mistakes",
    "Profile name can be overridden by IICS_PROFILE env var for CI/CD",
    "defaultProfile used when no --profile flag is given",
], Inches(0.5), Inches(6.0), Inches(12.3), Inches(1.3), font_size=14)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 8 – Session Caching
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Session Caching", "Authenticate once, reuse for 30 minutes")
footer(s)

add_text_box(s, "Official CLI – authenticates on every command",
             Inches(0.5), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=RED)
bullet_list(s, [
    "Login round-trip added to every command",
    "Risk of account lockout during scripted runs",
    "Slower in pipelines with many sequential commands",
], Inches(0.5), Inches(1.7), Inches(5.8), Inches(2.0), font_size=14, color=RED)

add_text_box(s, "iics-cli – session token cached with 30-min TTL",
             Inches(7.0), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=GREEN)
code_block(s, (
    "iics login                   # authenticates once\n"
    "iics user list               # reuses session\n"
    "iics role list               # reuses session\n"
    "iics connection list         # reuses session\n"
    "iics export create --name x  # reuses session\n\n"
    "# Separate cached session per profile\n"
    "iics user list --profile prod  # own token"
), Inches(7.0), Inches(1.7), Inches(5.8), Inches(2.8), font_size=11)

bullet_list(s, [
    "Sessions cached in ~/.iics/sessions.yaml",
    "Each profile has its own independent session entry",
    "Expired or invalid sessions trigger automatic re-login transparently",
    "401 responses auto-retry once with a fresh login – no manual intervention",
], Inches(0.5), Inches(4.1), Inches(12.3), Inches(2.5), font_size=15)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 9 – API Coverage
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Broad API Coverage", "20+ resource types vs. 5 operations in the official tool")
footer(s)

add_table(s,
    ["Resource", "Operations"],
    [
        ["objects",       "list, dependencies"],
        ["lookup",        "resolve IDs, names, paths"],
        ["connection",    "list, get, create, update, delete"],
        ["export",        "create, status, download"],
        ["import",        "upload, start, status, log"],
        ["schedule",      "list, get, create, update, delete"],
        ["project",       "create, update, delete"],
        ["folder",        "create, update, delete"],
        ["user",          "list, get, create, update, delete"],
        ["usergroup",     "list, get, create, update, delete"],
    ],
    Inches(0.4), Inches(1.25), Inches(6.0), Inches(5.7))

add_table(s,
    ["Resource", "Operations"],
    [
        ["role",          "list, get, create, update, delete"],
        ["privilege",     "list"],
        ["runtime",       "list, get, create, update"],
        ["agent",         "list, start, stop"],
        ["tag",           "assign, remove"],
        ["permission",    "get, set, delete"],
        ["securitylog",   "list"],
        ["metering",      "get, download"],
        ["sourcecontrol", "checkout, checkin, pull, commit"],
        ["state",         "fetch, load"],
    ],
    Inches(6.9), Inches(1.25), Inches(6.0), Inches(5.7))


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 10 – Region Support
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Comprehensive Region Support", "All IICS pod regions – not just us / eu / ap")
footer(s)

add_text_box(s, "Official CLI – 3 hardcoded regions",
             Inches(0.5), Inches(1.2), Inches(4.0), Inches(0.4),
             font_size=14, bold=True, color=RED)
bullet_list(s, ["-r us", "-r eu", "-r ap"],
            Inches(0.5), Inches(1.65), Inches(4.0), Inches(1.2),
            font_size=14, color=RED)

add_text_box(s, "iics-cli – full pod registry",
             Inches(5.0), Inches(1.2), Inches(7.8), Inches(0.4),
             font_size=14, bold=True, color=GREEN)

pods = [
    ["USW1, USW1-1, USW1-2", "US West"],
    ["USE2, USE4, USE6",      "US East"],
    ["USW3, USW3-1, USW5",   "US West (additional)"],
    ["CAC1",                  "Canada"],
    ["EMEA, EMWE1",           "Europe / Middle East / Africa"],
    ["APSE1, APJ, APNE1",    "Asia Pacific"],
]
add_table(s,
    ["Region Code(s)", "Geography"],
    pods,
    Inches(5.0), Inches(1.65), Inches(7.8), Inches(2.8))

add_text_box(s,
    "A loginUrl override is also supported for future pods or custom deployments.",
    Inches(0.5), Inches(5.6), Inches(12.3), Inches(0.4),
    font_size=14, italic=True, color=MID_BLUE)

code_block(s, (
    "# Use a built-in region\n"
    "iics user list --profile emea   # region: EMEA in config\n\n"
    "# Override with explicit login URL\n"
    "# loginUrl: https://dm-us.informaticacloud.com/ma/api/v2/user/login"
), Inches(0.5), Inches(4.3), Inches(12.3), Inches(1.2), font_size=11)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 11 – Single Binary
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Single Binary – No JVM Required", "Go binary vs. Java application")
footer(s)

add_text_box(s, "Official CLI – Java application",
             Inches(0.5), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=RED)
bullet_list(s, [
    "Requires JRE installed on every machine",
    "JVM startup overhead on every invocation",
    "Larger Docker images (JRE ~200-400 MB)",
    "Version conflicts between projects",
    "Complex installation procedure",
], Inches(0.5), Inches(1.7), Inches(5.8), Inches(3.5), font_size=15, color=RED)

add_text_box(s, "iics-cli – compiled Go binary",
             Inches(7.0), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=GREEN)
bullet_list(s, [
    "Single statically compiled binary – no dependencies",
    "Linux, macOS, Windows (amd64 and arm64)",
    "Copy to PATH and run immediately",
    "~8 MB binary – tiny Docker layer",
    "Instant startup – no JVM warmup",
], Inches(7.0), Inches(1.7), Inches(5.8), Inches(3.5), font_size=15, color=GREEN)

code_block(s, (
    "# Install – one line\n"
    "curl -L https://github.com/jbrazda/iics-cli/releases/latest/download/iics_linux_amd64 \\\n"
    "  -o /usr/local/bin/iics && chmod +x /usr/local/bin/iics\n\n"
    "# Docker – minimal image\n"
    "COPY iics /usr/local/bin/iics\n"
    "# No apt-get install default-jre needed"
), Inches(0.5), Inches(5.3), Inches(12.3), Inches(1.8), font_size=11)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 12 – Structured Commands + Help
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Structured Subcommands & Built-in Help", "kubectl / gh / aws style resource-verb hierarchy")
footer(s)

code_block(s, (
    "# Resource  Verb  Flags\n"
    "iics user   list\n"
    "iics user   get   --id abc123\n"
    "iics user   create --from-file user.json\n\n"
    "iics connection list\n"
    "iics connection update --id xyz --from-file conn.json\n\n"
    "iics export create --name nightly --project Prod\n"
    "iics export status --job-id 12345\n"
    "iics export download --job-id 12345 --output export.zip\n\n"
    "iics sourcecontrol checkin --project Prod --comment \"release 2.4\"\n"
    "iics state fetch --project Prod --output state.json"
), Inches(0.5), Inches(1.3), Inches(7.5), Inches(5.5), font_size=11)

add_text_box(s, "Built-in help at every level",
             Inches(8.2), Inches(1.3), Inches(4.8), Inches(0.4),
             font_size=14, bold=True, color=DARK_BLUE)
code_block(s, (
    "iics --help\n\n"
    "iics user --help\n\n"
    "iics export create --help"
), Inches(8.2), Inches(1.75), Inches(4.8), Inches(1.5), font_size=11)

bullet_list(s, [
    "Same pattern as kubectl, gh, aws – familiar to DevOps teams",
    "Tab completion supported",
    "Per-command docs in docs/documentation/",
    "Full flag reference in built-in --help",
], Inches(8.2), Inches(3.4), Inches(4.8), Inches(2.5), font_size=13)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 13 – Error Reporting
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Better Error Reporting", "Structured API errors with HTTP context – all on stderr")
footer(s)

add_text_box(s, "Official CLI – vague error messages",
             Inches(0.5), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=RED)
code_block(s, (
    "$ iics export -u user -p pass -r us \\\n"
    "  -a \"Project/Missing.Process\"\n\n"
    "Error occurred.\n"
    "exit code: 0"
), Inches(0.5), Inches(1.65), Inches(5.8), Inches(1.5), font_size=11)

add_text_box(s, "iics-cli – structured errors to stderr",
             Inches(7.0), Inches(1.2), Inches(5.8), Inches(0.4),
             font_size=14, bold=True, color=GREEN)
code_block(s, (
    "$ iics export create --name bad --project X\n\n"
    "Error: API error 400: project 'X' not found\n"
    "  HTTP 400 Bad Request\n"
    "  {\n"
    "    \"error\": {\n"
    "      \"code\": \"ICS-012345\",\n"
    "      \"message\": \"project 'X' not found\"\n"
    "    }\n"
    "  }\n"
    "exit code: 1"
), Inches(7.0), Inches(1.65), Inches(5.8), Inches(2.5), font_size=11)

bullet_list(s, [
    "Error summary always on stderr – stdout stays clean for piping",
    "--debug flag prints full request body for deep troubleshooting",
    "--verbose flag enables extra HTTP detail",
    "API error code, HTTP status, and response body all included",
    "Usage errors (exit 2) print a hint without cluttering stdout",
], Inches(0.5), Inches(4.4), Inches(12.3), Inches(2.5), font_size=15)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 14 – Feature Comparison Summary
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Feature Comparison Summary")
footer(s)

add_table(s,
    ["Capability", "Informatica CLI V2", "iics-cli"],
    [
        ["Actively maintained",              "No",      "Yes"],
        ["POSIX exit codes",                 "No",      "Yes"],
        ["JSON / CSV output",                "No",      "Yes"],
        ["Pipe-friendly stdout/stderr split","No",      "Yes"],
        ["Credentials in config file",       "No",      "Yes"],
        ["Named environment profiles",       "No",      "Yes"],
        ["Session caching",                  "No",      "Yes"],
        ["User & group management",          "No",      "Yes"],
        ["Connection CRUD",                  "No",      "Yes"],
        ["Schedule CRUD",                    "No",      "Yes"],
        ["Agent & runtime management",       "No",      "Yes"],
        ["Source control operations",        "No",      "Yes"],
        ["All IICS regions",                 "No",      "Yes"],
        ["Single binary – no JVM",           "No",      "Yes"],
        ["Built-in contextual help",         "Minimal", "Yes"],
    ],
    Inches(0.4), Inches(1.2), Inches(12.5), Inches(5.95))


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 15 – Getting Started
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
slide_header(s, "Getting Started", "Install, configure, and run your first command in minutes")
footer(s)

code_block(s, (
    "# 1. Download the binary for your platform\n"
    "curl -L https://github.com/jbrazda/iics-cli/releases/latest/download/iics_linux_amd64 \\\n"
    "  -o /usr/local/bin/iics && chmod +x /usr/local/bin/iics\n\n"
    "# 2. Create ~/.iics/config.yaml\n"
    "mkdir -p ~/.iics && chmod 700 ~/.iics\n"
    "cat > ~/.iics/config.yaml << 'EOF'\n"
    "defaultProfile: default\n"
    "profiles:\n"
    "  default:\n"
    "    region: US\n"
    "    username: your-email@company.com\n"
    "EOF\n"
    "chmod 600 ~/.iics/config.yaml\n\n"
    "# 3. Set your password in the environment\n"
    "export IICS_PASSWORD='your-password'\n\n"
    "# 4. Login (optional – cached for 30 min)\n"
    "iics login\n\n"
    "# 5. Start exploring\n"
    "iics user list\n"
    "iics connection list -o json | jq '.[].name'\n"
    "iics export create --name my-first-export --project MyProject"
), Inches(0.5), Inches(1.3), Inches(12.3), Inches(5.8), font_size=11)


# ─────────────────────────────────────────────────────────────────────────────
# SLIDE 16 – Thank You / Questions
# ─────────────────────────────────────────────────────────────────────────────
s = prs.slides.add_slide(blank_layout)
add_rect(s, 0, 0, SLIDE_W, SLIDE_H, DARK_BLUE)
add_rect(s, 0, Inches(3.5), SLIDE_W, Inches(0.06), ORANGE)

add_text_box(s, "Questions?",
             Inches(1), Inches(1.5), Inches(11), Inches(1.0),
             font_size=54, bold=True, color=WHITE, align=PP_ALIGN.CENTER)

add_text_box(s, "github.com/jbrazda/iics-cli",
             Inches(1), Inches(2.8), Inches(11), Inches(0.6),
             font_size=22, color=RGBColor(0xFF, 0xAA, 0x44), align=PP_ALIGN.CENTER)

add_text_box(s, "Source code  •  Releases  •  Issues  •  Documentation",
             Inches(1), Inches(3.6), Inches(11), Inches(0.5),
             font_size=16, color=RGBColor(0xBB, 0xCC, 0xFF), align=PP_ALIGN.CENTER)

add_text_box(s, "docs/documentation/  in the repository for full per-command reference",
             Inches(1), Inches(4.3), Inches(11), Inches(0.5),
             font_size=14, italic=True, color=MID_GRAY, align=PP_ALIGN.CENTER)


# ─────────────────────────────────────────────────────────────────────────────
# Save
# ─────────────────────────────────────────────────────────────────────────────
out = "iics-cli-presentation.pptx"
prs.save(out)
print(f"Saved: {out}")
