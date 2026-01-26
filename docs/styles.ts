type Theme = {
  /** Default Background */
  base00: string,
  /** Lighter Background (Used for status bars, line number and folding marks) */
  base01: string,
  /** Selection Background */
  base02: string,
  /** Comments, Invisibles, Line Highlighting */
  base03: string,
  /** Dark Foreground (Used for status bars) */
  base04: string,
  /** Default Foreground, Caret, Delimiters, Operators */
  base05: string,
  /** Light Foreground (Not often used) */
  base06: string,
  /** Light Background (Not often used) */
  base07: string,
  /** Variables, XML Tags, Markup Link Text, Markup Lists, Diff Deleted */
  base08: string,
  /** Integers, Boolean, Constants, XML Attributes, Markup Link Url */
  base09: string,
  /** Classes, Markup Bold, Search Text Background */
  base0A: string,
  /** Strings, Inherited Class, Markup Code, Diff Inserted */
  base0B: string,
  /** Support, Regular Expressions, Escape Characters, Markup Quotes */
  base0C: string,
  /** Functions, Methods, Attribute IDs, Headings */
  base0D: string,
  /** Keywords, Storage, Selector, Markup Italic, Diff Changed */
  base0E: string,
  /** Deprecated, Opening/Closing Embedded Language Tags, e.g. <?php ?> */
  base0F: string,
};

import simpletheme from "./themes/simple.ts";
import yorhatheme from "./themes/yorha.ts";

const themes: Record<string, Theme> = {
  // "purpledream": importstr "themes/purpledream.yaml",
  // "synth-midnight-dark": importstr "themes/synth-midnight-dark.yaml",
  // "synth-midnight-light": importstr "themes/synth-midnight-light.yaml",
  // "windows-95": importstr "themes/windows-95.yaml",
  "simple": simpletheme,
  "yorha": yorhatheme,
};
const theme = "simple";
const colors = Object.fromEntries(Object.entries(themes[theme]).map(([k, v]) => [k, "#"+v]));

export default [
  ["*, *:before, *:after", {
    "box-sizing": "inherit",
    "font-size": "inherit",
  }],
  [".markdown-section", {
    "position": "relative",
    "max-width": "95%",
    "margin": "0 auto",
  }],
  [".markdown-section a", {
    "overflow-wrap": "anywhere",
    "border-bottom": "var(--link-border-bottom)",
    "color": "var(--link-color)",
    "text-decoration": "underline",
  }],
  [".markdown-section a.anchor", {
    "border-bottom": "0",
    "color": "inherit",
    "text-decoration": "none",
  }],
  [".markdown-section a.anchor:hover", {
    "text-decoration": "underline",
    "color": "inherit",
  }],
  [".markdown-section a:hover", {
    "border-bottom": "var(--link-border-bottom--hover, var(--link-border-bottom, 0))",
    "color": "var(--link-color--hover, var(--link-color))",
  }],
  [".markdown-section code, .markdown-section pre", {
    "border-radius": "var(--code-block-border-radius)",
    "font-family": "var(--code-font-family)",
    "font-size": "var(--code-font-size)",
    "font-weight": "var(--code-font-weight)",
    "letter-spacing": "normal",
    "line-height": "1.2em",
    "tab-size": "var(--code-tab-size)",
    "text-align": "left",
    "white-space": "pre",
    "word-spacing": "normal",
    "word-wrap": "normal",
    "word-break": "normal",
    "hyphens": "none",
    "position": "relative",
    "margin": "var(--code-block-margin)",
    "padding": "0",
    "overflow-wrap": "anywhere",
  }],
  [".markdown-section code:not([class*=lang-]):not([class*=language-])", {
    "margin": "var(--code-inline-margin)",
    "padding": "var(--code-inline-padding)",
    "border-radius": "var(--code-inline-border-radius)",
    "background": "var(--code-inline-background)",
    "color": "var(--code-inline-color, currentColor)",
    "white-space": "nowrap",
  }],
  [".markdown-section h1", {
    "margin": "1rem 0rem -0.5rem 0rem",
    "font-size": "var(--heading-h1-font-size)",
    "font-weight": "400",
    "color": "var(--heading-color)",
    "line-height": "4.00rem",
  }],
  [[[
    ".markdown-section h1 a",
    ".markdown-section h2 a",
    ".markdown-section h3 a",
    ".markdown-section h4 a",
    ".markdown-section h5 a",
    ".markdown-section h6 a"].join(", ")], {
    "display": "inline-block",
  }],
  [[[
    ".markdown-section h1 code",
    ".markdown-section h2 code",
    ".markdown-section h3 code",
    ".markdown-section h4 code",
    ".markdown-section h5 code",
    ".markdown-section h6 code"].join(", ")], {
    "font-size": ".875em",
  }],
  [[[
    ".markdown-section h1+h2",
    ".markdown-section h1+h3",
    ".markdown-section h1+h4",
    ".markdown-section h1+h5",
    ".markdown-section h1+h6",
    ".markdown-section h2+h3",
    ".markdown-section h2+h4",
    ".markdown-section h2+h5",
    ".markdown-section h2+h6",
    ".markdown-section h3+h4",
    ".markdown-section h3+h5",
    ".markdown-section h3+h6",
    ".markdown-section h4+h5",
    ".markdown-section h4+h6",
    ".markdown-section h5+h6"].join(", ")], {
    "margin-top": "1rem",
  }],
  [[[
    ".markdown-section h1:first-child",
    ".markdown-section h2:first-child",
    ".markdown-section h3:first-child",
    ".markdown-section h4:first-child",
    ".markdown-section h5:first-child",
    ".markdown-section h6:first-child"].join(", ")], {
    "margin-top": "0",
  }],
  [".markdown-section h2", {
    "margin": "2.5rem 0 1rem",
    "padding": "0 0 0.75rem 0",
    "border-width": "0 0 1px 0",
    "border-style": "solid",
    "border-color": colors.base07,
    "font-size": "var(--heading-h2-font-size)",
    "font-weight": "400",
    "color": "var(--heading-color)",
    "line-height": "1.55rem",
  }],
  [".markdown-section h3", {
    "margin": "1rem 0rem -.5rem 0rem",
    "font-size": "var(--heading-h3-font-size)",
    "font-weight": "400",
    "color": "var(--heading-color)",
    "line-height": "1.35rem",
  }],
  [".markdown-section h4", {
    "margin": "1rem 0rem -.5rem 0rem",
    "font-size": "var(--heading-h4-font-size)",
    "font-weight": "400",
    "color": "var(--heading-color)",
    "line-height": "1.30rem",
  }],
  [".markdown-section h5", {
    "margin": "1rem 0rem -.5rem 0rem",
    "font-size": "var(--heading-h5-font-size)",
    "font-weight": "400",
    "color": "var(--heading-color)",
    "line-height": "1.25rem",
  }],
  [".markdown-section h6", {
    "margin": "1rem 0rem 0rem 0rem",
    "font-size": "var(--heading-h6-font-size)",
    "font-weight": "400",
    "color": "var(--heading-color)",
    "line-height": "1.20rem",
  }],
  [".markdown-section img", {
    "max-width": "100%",
  }],
  [".markdown-section ol ol, .markdown-section ol ul, .markdown-section ul ol, .markdown-section ul ul", {
    "margin-top": ".15rem",
    "margin-bottom": ".15rem",
  }],
  [".markdown-section ol, .markdown-section ul", {
    "padding-left": "1.5rem",
  }],
  [".markdown-section p, .markdown-section ol, .markdown-section ul", {
    "margin": "1em 0",
  }],
  [".markdown-section pre code", {
    "display": "block",
    "overflow": "auto",
    "padding": "0.5em 1em",
    "word-break": "break-all",
  }],
  [".markdown-section pre[data-lang]::after", {
    "content": "attr(data-lang)",
    "position": "absolute",
    "top": ".75em",
    "right": ".75em",
    "opacity": ".6",
    "color": "inherit",
    "font-size": "var(--font-size-s)",
    "line-height": "1",
  }],
  [".markdown-section ul", {
    "list-style": "square inside none",
  }],
  [".sidebar", {
    // "visibility": "hidden",
    "display": "flex",
    "flex-direction": "column",
    // "position": "fixed",
    "z-index": "10",
    "top": "0",
    "right": "100%",
    "overflow-x": "hidden",
    "overflow-y": "auto",
    "height": "100vh",
    "width": "var(--sidebar-width)",
    "padding": "var(--sidebar-padding)",
    "border-width": "var(--sidebar-border-width)",
    "border-style": "solid",
    "border-color": "var(--sidebar-border-color)",
    "background": "var(--sidebar-background)",

    //   @media(min-width: 48em) { .sidebar { visibility: visible; position: absolute; transform: translateX(var(--sidebar-width)) } }
    "visibility": "visible",
    "position": "absolute",
    "transform": "translateX(var(--sidebar-width))",
  }],
  [".sidebar+.content", { "margin-left": "var(--sidebar-width)" }], // @media(min-width: 48em) { .sidebar+.content { margin-left: var(--sidebar-width) } }
  [".sidebar-nav", {
    "order": "1",
    "margin": "var(--sidebar-nav-margin)",
    "padding": "var(--sidebar-nav-padding)",
    "background": "var(--sidebar-nav-background)",
  }],
  [".sidebar-nav a", {
    "display": "block",
    "margin": "var(--sidebar-nav-link-margin)",
    "padding": "var(--sidebar-nav-link-padding)",
    "border-width": "var(--sidebar-nav-link-border-width, 0)",
    "border-style": "var(--sidebar-nav-link-border-style)",
    "border-color": "var(--sidebar-nav-link-border-color)",
    "border-radius": "var(--sidebar-nav-link-border-radius)",
    "background": "var(--sidebar-nav-link-background)",
    "color": "var(--sidebar-nav-link-color)",
    "font-weight": "var(--sidebar-nav-link-font-weight)",
    "text-decoration": "var(--sidebar-nav-link-text-decoration)",
    "text-decoration-color": "var(--sidebar-nav-link-text-decoration-color)",
    "text-overflow": "ellipsis",
    "overflow": "visible",
    "white-space": "break-spaces",
  }],
  [".sidebar-nav a img", {
    "margin-top": "-0.25em",
    "vertical-align": "middle",
  }],
  [".sidebar-nav a img:first-child", {
    "margin-right": ".5em",
  }],
  [".sidebar-nav a:hover", {
    "border-width": "var(--sidebar-nav-link-border-width--hover, var(--sidebar-nav-link-border-width, 0))",
    "border-style": "var(--sidebar-nav-link-border-style--hover, var(--sidebar-nav-link-border-style))",
    "border-color": "var(--sidebar-nav-link-border-color--hover, var(--sidebar-nav-link-border-color))",
    "background": "var(--sidebar-nav-link-background--hover, var(--sidebar-nav-link-background))",
    "color": "var(--sidebar-nav-link-color--hover, var(--sidebar-nav-link-color))",
    "font-weight": "var(--sidebar-nav-link-font-weight--hover, var(--sidebar-nav-link-font-weight))",
    "text-decoration": "var(--sidebar-nav-link-text-decoration--hover, var(--sidebar-nav-link-text-decoration))",
    "text-decoration-color": "var(--sidebar-nav-link-text-decoration-color)",
  }],
  [".sidebar-nav li>a", {
    "background-repeat": "var(--sidebar-nav-pagelink-background-repeat)",
    "background-size": "var(--sidebar-nav-pagelink-background-size)",
  }],
  [".sidebar-nav li>a:before", {
    "display": "inline-block",
  }],
  ['.sidebar-nav li>a[href^="/"], .sidebar-nav li>a[href^="#/"]', {
    "background": "var(--sidebar-nav-pagelink-background)",
  }],
  ['.sidebar-nav li>a[href^="/"], .sidebar-nav li>a[href^="/"]~ul a, .sidebar-nav li>a[href^="#/"], .sidebar-nav li>a[href^="#/"]~ul a', {
    "padding": "var(--sidebar-nav-pagelink-padding, var(--sidebar-nav-link-padding))",
  }],
  [".sidebar-nav ul", {
    "margin": "0",
    "padding": "0",
    "list-style": "none",
  }],
  [".sidebar-nav ul ul", {
    "margin-left": "var(--sidebar-nav-indent)",
  }],
  [[
    ".sidebar-nav ul>li:first-child>span",
    ".sidebar-nav ul>li:first-child>strong",
  ].join(",\n  "), {
    "margin-top": "0",
  }],
  [".sidebar-nav ul>li>span", {
    "display": "block",
    "margin": "var(--sidebar-nav-strong-margin)",
    "padding": "var(--sidebar-nav-strong-padding)",
    "border-width": "var(--sidebar-nav-strong-border-width, 0)",
    "border-style": "solid",
    "border-color": "var(--sidebar-nav-strong-border-color)",
    "color": "var(--sidebar-nav-strong-color)",
    "font-size": "var(--sidebar-nav-strong-font-size)",
    "font-weight": "var(--sidebar-nav-strong-font-weight)",
    "text-transform": "var(--sidebar-nav-strong-text-transform)",
  }],
  [".sidebar-nav ul>li>span+ul", {
    "margin-left": "0",
  }],
  [".sidebar-nav>:last-child", {
    "margin-bottom": "2rem",
  }],
  [".sidebar-nav>ul>li>a:before", {
    "content": "var(--sidebar-nav-link-before-content-l1, var(--sidebar-nav-link-before-content))",
    "margin": "var(--sidebar-nav-link-before-margin-l1, var(--sidebar-nav-link-before-margin))",
    "color": "var(--sidebar-nav-link-before-color-l1, var(--sidebar-nav-link-before-color))",
  }],
  [".sidebar-nav>ul>li>ul>li>a:before", {
    "content": "var(--sidebar-nav-link-before-content-l2, var(--sidebar-nav-link-before-content))",
    "margin": "var(--sidebar-nav-link-before-margin-l2, var(--sidebar-nav-link-before-margin))",
    "color": "var(--sidebar-nav-link-before-color-l2, var(--sidebar-nav-link-before-color))",
  }],
  [".sidebar-nav>ul>li>ul>li>ul>li>a:before", {
    "content": "var(--sidebar-nav-link-before-content-l3, var(--sidebar-nav-link-before-content))",
    "margin": "var(--sidebar-nav-link-before-margin-l3, var(--sidebar-nav-link-before-margin))",
    "color": "var(--sidebar-nav-link-before-color-l3, var(--sidebar-nav-link-before-color))",
  }],
  [".sidebar-nav>ul>li>ul>li>ul>li>ul>li>a:before", {
    "content": "var(--sidebar-nav-link-before-content-l4, var(--sidebar-nav-link-before-content))",
    "margin": "var(--sidebar-nav-link-before-margin-l4, var(--sidebar-nav-link-before-margin))",
    "color": "var(--sidebar-nav-link-before-color-l4, var(--sidebar-nav-link-before-color))",
  }],
  [".sidebar>h1", {
    "margin": "0",
    "padding": "var(--sidebar-name-padding)",
    "background": "var(--sidebar-name-background)",
    "color": "var(--sidebar-name-color)",
    "font-family": "var(--sidebar-name-font-family)",
    "font-size": "var(--sidebar-name-font-size)",
    "font-weight": "var(--sidebar-name-font-weight)",
    "text-align": "var(--sidebar-name-text-align)",
  }],
  [".sidebar>h1 img", {
    "max-width": "100%",
  }],
  ["::selection", {
    "background": "var(--selection-color)",
  }],
  [":root", {
    "--mono-hue": "113",
    "--mono-saturation": "0%",
    "--mono-shade3": "hsl(var(--mono-hue), var(--mono-saturation), 20%)",
    "--mono-shade2": "hsl(var(--mono-hue), var(--mono-saturation), 30%)",
    "--mono-shade1": "hsl(var(--mono-hue), var(--mono-saturation), 40%)",
    "--mono-base":   "hsl(var(--mono-hue), var(--mono-saturation), 50%)",
    "--mono-tint1":  "hsl(var(--mono-hue), var(--mono-saturation), 70%)",
    "--mono-tint2":  "hsl(var(--mono-hue), var(--mono-saturation), 89%)",
    "--mono-tint3":  "hsl(var(--mono-hue), var(--mono-saturation), 97%)",
    "--theme-color": "hsl(204, 90%, 45%)",
    "--link-color": colors.base09,
    "--selection-color": colors.base02,
    "--base-background-color": colors.base00,
    "--base-color": colors.base05,
    "--heading-color": colors.base0D,//"var(--mono-shade3)",

    "--base-font-size": "1.125rem",
    "--base-font-weight": "normal",
    "--base-line-height": "1.6",

    "--heading-h1-font-size": "var(--font-size-xxl)",
    "--heading-h2-font-size": "var(--font-size-xl)",
    "--heading-h3-font-size": "var(--font-size-l)",
    "--heading-h4-font-size": "var(--font-size-m)",
    "--heading-h5-font-size": "var(--font-size-s)",
    "--heading-h6-font-size": "var(--font-size-xs)",

    "box-sizing": "border-box",
    "background-color": "var(--base-background-color)",
    "font-size": "var(--base-font-size)",
    "font-weight": "var(--base-font-weight)",
    "line-height": "var(--base-line-height)",
    "letter-spacing": "var(--base-letter-spacing)",
    "color": "var(--base-color)",

    "--modular-scale": "1.333",
    "--modular-scale--2": "calc(var(--modular-scale--1) / var(--modular-scale))",
    "--modular-scale--1": "calc(var(--modular-scale-1) / var(--modular-scale))",
    "--modular-scale-1":  "1rem",
    "--modular-scale-2":  "calc(var(--modular-scale-1) * var(--modular-scale))",
    "--modular-scale-3":  "calc(var(--modular-scale-2) * var(--modular-scale))",
    "--modular-scale-4":  "calc(var(--modular-scale-3) * var(--modular-scale))",
    "--font-size-xxl": "var(--modular-scale-4)",
    "--font-size-xl":  "var(--modular-scale-3)",
    "--font-size-l":   "var(--modular-scale-2)",
    "--font-size-m":   "var(--modular-scale-1)",
    "--font-size-s":   "var(--modular-scale--1)",
    "--font-size-xs":  "var(--modular-scale--2)",

    "--hr-border": "1px solid var(--mono-tint2)",
    "--pre-font-size": "var(--code-font-size)",
    "--small-font-size": "var(--font-size-s)",
    "--strong-color": "var(--heading-color)",
    "--strong-font-weight": "600",
    "--subsup-font-size": "var(--font-size-s)",

    "--navbar-root-border-style": "solid",
    "--navbar-menu-background": "var(--base-background-color)",
    "--navbar-root-color--active": "var(--theme-color)",
    "--navbar-menu-box-shadow": "rgba(45, 45, 45, 0.05) 0px 0px 1px, rgba(49, 49, 49, 0.05) 0px 1px 2px, rgba(42, 42, 42, 0.05) 0px 2px 4px, rgba(32, 32, 32, 0.05) 0px 4px 8px, rgba(49, 49, 49, 0.05) 0px 8px 16px, rgba(35, 35, 35, 0.05) 0px 16px 32px",
    "--navbar-menu-padding": "0.5em",
    "--navbar-menu-link-border-style": "solid",
    "--navbar-menu-link-margin": "0.75em 0.5em",
    "--navbar-menu-link-padding": "0.2em 0",

    // "--sidebar-name-color": "#0374B5",
    // "--sidebar-background": "var(--base-background-color)",
    "--sidebar-background": colors.base01,
    "--sidebar-padding": "0 25px",
    "--sidebar-width": "17rem",
    "--sidebar-border-color": colors.base07,
    "--sidebar-border-width": "0 1px 0 0",
    "--sidebar-name-color": "var(--theme-color)",
    "--sidebar-name-font-weight": "300",
    // "--sidebar-name-font-weight": "normal",
    "--sidebar-name-font-size": "var(--font-size-l)",
    "--sidebar-name-margin": "1.5rem 0 0",
    "--sidebar-name-text-align": "center",
    "--sidebar-nav-strong-border-color": colors.base07,
    "--sidebar-nav-strong-color": "var(--heading-color)",
    "--sidebar-nav-strong-font-weight": "var(--strong-font-weight)",
    "--sidebar-nav-strong-margin": "1.5em 0 0.5em",
    "--sidebar-nav-strong-padding": "0.25em 0",
    // "--sidebar-nav-strong-margin": "2em -25px 0.75em 0",
    // "--sidebar-nav-strong-padding": "0.25em 0 0.75em 0",
    "--sidebar-nav-indent": "1em",
    "--sidebar-nav-margin": "1.5rem 0 0",
    "--sidebar-nav-link-border-style": "solid",
    // "--sidebar-nav-link-border-width": "0",
    "--sidebar-nav-link-border-color": "transparent",
    "--sidebar-nav-link-border-color--active": "#0374B5",
    // "--sidebar-nav-link-border-color--active": "var(--theme-color)",
    "--sidebar-nav-link-border-width": "0 4px 0 0",
    "--sidebar-nav-link-color": "var(--base-color)",
    "--sidebar-nav-link-font-weight": "normal",
    "--sidebar-nav-link-padding": "0.25em 0",
    // "--sidebar-nav-link-text-decoration--active": "underline",
    "--sidebar-nav-link-text-decoration--hover": "underline",
    "--sidebar-nav-link-text-decoration": "none",
    "--sidebar-nav-link-text-decoration--active": "none",
    "--sidebar-nav-link-before-margin": "0 0.35em 0 0",
    // "--sidebar-nav-link-color--active": "#0374B5",
    "--sidebar-nav-link-color--active": "var(--theme-color)",
    "--sidebar-nav-link-font-weight--active": "bold",
    "--sidebar-nav-link-margin": "0 -25px 0 0",
    "--sidebar-nav-pagelink-background": "no-repeat 2px calc(50% - 2.5px) / 6px 5px linear-gradient(45deg, transparent 2.75px, var(--mono-tint1) 2.75px 4.25px, transparent 4px), no-repeat 2px calc(50% + 2.5px) / 6px 5px linear-gradient(135deg, transparent 2.75px, var(--mono-tint1) 2.75px 4.25px, transparent 4px)",
    "--sidebar-nav-pagelink-background--active": "no-repeat 0px center / 5px 6px linear-gradient(225deg, transparent 2.75px, var(--theme-color) 2.75px 4.25px, transparent 4.25px), no-repeat 5px center / 5px 6px linear-gradient(135deg, transparent 2.75px, var(--theme-color) 2.75px 4.25px, transparent 4.25px)",
    "--sidebar-nav-pagelink-background--collapse": "no-repeat 2px calc(50% - 2.5px) / 6px 5px linear-gradient(45deg, transparent 2.75px, var(--theme-color) 2.75px 4.25px, transparent 4px), no-repeat 2px calc(50% + 2.5px) / 6px 5px linear-gradient(135deg, transparent 2.75px, var(--theme-color) 2.75px 4.25px, transparent 4px)",
    "--sidebar-nav-pagelink-background--loaded": "no-repeat 0px center / 5px 6px linear-gradient(225deg, transparent 2.75px, var(--mono-tint1) 2.75px 4.25px, transparent 4.25px), no-repeat 5px center / 5px 6px linear-gradient(135deg, transparent 2.75px, var(--mono-tint1) 2.75px 4.25px, transparent 4.25px)",
    "--sidebar-nav-pagelink-padding": "0.25em 0 0.25em 20px",
    "--sidebar-nav-strong-border-width": "0 0 1px 0",
    "--sidebar-nav-strong-font-size": "smaller",
    "--sidebar-nav-strong-text-transform": "uppercase",

    "--code-theme-background": colors.base01,
    "--code-theme-comment": colors.base03,
    "--code-theme-function": colors.base0D,
    "--code-theme-keyword": colors.base0E,
    "--code-theme-operator": colors.base05,
    "--code-theme-punctuation": colors.base0F,
    "--code-theme-selector": colors.base0E,
    "--code-theme-tag": colors.base08,
    "--code-theme-text": colors.base0B,
    "--code-theme-variable": colors.base08,
    "--code-font-family": 'Inconsolata, Consolas, Menlo, Monaco, "Andale Mono WT", "Andale Mono", "Lucida Console", "DejaVu Sans Mono", "Bitstream Vera Sans Mono", "Courier New", Courier, monospace',
    "--code-font-size": "calc(var(--font-size-m) * 0.95)",
    "--code-font-weight": "normal",
    "--code-tab-size": "4",
    "--code-block-border-radius": "var(--border-radius-m)",
    "--code-block-margin": "1em 0",
    "--code-inline-background": colors.base07,
    "--code-inline-border-radius": "var(--border-radius-s)",
    "--code-inline-color": "var(--code-theme-text)",
    "--code-inline-margin": "0 0.15em",
    "--code-inline-padding": "0.125em 0.4em",

    "--border-radius-s": "2px",
    "--border-radius-m": "4px",
    "--border-radius-l": "8px",
  }],
  ["a", {
    "text-decoration": "none",
    "text-decoration-skip-ink": "auto",
  }],
  ["body *", {
    "scrollbar-color": "hsla(var(--mono-hue), var(--mono-saturation), 50%, 0.3) hsla(var(--mono-hue), var(--mono-saturation), 50%, 0.1)",
    "scrollbar-width": "thin",
  }],
  ["body .sidebar", {
    "padding": "var(--sidebar-padding)",
  }],
  ["body.sticky .sidebar", {
    "position": "fixed",
  }],
  ["code[class*=lang-], pre[data-lang]", {
    "color": "var(--code-theme-text)",
  }],
  ["hr", {
    "height": "0",
    "margin": "2em 0",
    "border": "none",
    "border-bottom": "var(--hr-border, 0)",
  }],
  ["html", {
    "font-family": '"Source Sans Pro", "Helvetica Neue", Arial, sans-serif',
  }],
  ["main", {
    "display": "block",
    "position": "relative",
    "overflow-x": "hidden",
    "min-height": "100vh",
  }],
  ["pre", {
    "font-family": "var(--code-font-family)",
    "font-size": "var(--pre-font-size)",
    "font-weight": "normal",
  }],
  ["pre[data-lang]::selection, code[class*=lang-]::selection", {
    "background": "var(--code-theme-selection, var(--selection-color))",
  }],
  // ["table", {
  //   "border-spacing": "0",
  // }],
  // ["th", {
  //   "border-bottom": "0.1rem solid var(--sidebar-border-color)",
  // }],
  // ["body", {
  //   "background-image": "linear-gradient(to right, #ccc8b1 1px, rgba(204,200,177,0) 1px), linear-gradient(to bottom, #ccc8b1 1px, rgba(204,200,177,0) 1px)",
  //   "background-size": "0.3rem 0.3rem",
  // }],
  // @media(min-width: 48em) { body.sticky .sidebar { position: fixed } }
//   @media print {
//     .sidebar {
//       display: none
//     }
//   }
] as [string, Record<string, string>][];
