import {styleText} from "util";
import {join} from "path";
import {stat} from "fs/promises";
import {deflate} from "pako";

function dedent(s: string): string {
  const lines = s.substring(1).trimEnd().split("\n");

  const minIndent = Math.min(...lines.filter(s => s.trim() !== "").map(line => line.length - line.trimStart().length));
  if (!isFinite(minIndent) || minIndent === 0)
    return s;

  return lines
    .map(line => line.slice(minIndent))
    .join("\n")
    .trim();
}

function manifestXmlJsonml(x: HTMLNode): string {
  if (typeof x === "string")
    return x;

  const [tag, attrs, ...children] = x;
  const props = Object
    .keys(attrs)
    .toSorted()
    .map(k => ` ${k}="${attrs[k]}"`)
    .join("");
  const content = children
    .map(manifestXmlJsonml)
    .filter(s => s !== "")
    .join("");
  return `<${tag}${props}>${content}</${tag}>`;
}

const std = {
  // string utils
  substr: (s: string, start: number, end: number) => s.substring(start, end),
  findSubstr: (pattern: string, s: string): number[] => {
    const indices: number[] = [];
    let i = s.indexOf(pattern);
    while (i != -1) {
      indices.push(i);
      i = s.indexOf(pattern, i + 1);
    }
    return indices;
  },
  // list utils
  joinList: <T>(sep: T[], xs: T[][]): T[] => xs.flatMap((x, i) => [...(i > 0 ? sep : []), ...x]),
};

type Lang = "sh" | "jsonnet" | "yaml" | "toml" | "ini" | "hcl" | "json";

type Adapter<T, X> = {
  render: (doc: <R, Y>(a: Adapter<R, Y>) => R[]) => X,
  h1: (title: string) => T,
  h2: (title: string) => T,
  h3: (title: string) => T,
  tabs: (tabs: [string, ...T[]][]) => T,
  p: (...xs: (string | T)[]) => T,
  b: (s: string) => T,
  a: (text: string, href: string) => T,
  a_external: (text: string, href: string) => T,
  code: (code: string) => T,
  codeblock: (code: string, lang: Lang) => T,
  ul: (...xs: (string | T)[][]) => T,
  icon: () => T,
  table: (headers: string[], ...xs: (string | T)[][]) => T,
  process_state_diagram: (source: string) => T,
};

type TOCItem = {title: string, level: 0|1|2|3};
type TOCItem2 = {title: string, children: TOCItem2[]};
const toc: Adapter<TOCItem | [], TOCItem2[]> & {compose: (xs: (TOCItem | [])[]) => TOCItem2[]} = {
  render: (doc): TOCItem2[] => toc.compose(doc(toc)),
  compose: (xs: (TOCItem | [])[]): TOCItem2[] => xs.reduce((acc: TOCItem2[], x: (TOCItem | [])) => {
    const node = (x: TOCItem) => ({title: x.title, children: []});
    if (Array.isArray(x)) return acc;
    else if (x.level == 0) return [...acc, node(x)];
    else if (x.level == 1) {
      const n = acc.length;
      const last = acc[n-1];
      return [...acc.slice(0, n-1), {title: last.title, children: [...last.children, node(x)]}];
    } else if (x.level == 2) {
      const n = acc.length;
      const last = acc[n-1];
      const m = last.children.length;
      const lastlast = last.children[m-1];
      return [...acc.slice(0, n-1), {
        title: last.title,
        children: [...last.children.slice(0, m-1), {
          title: lastlast.title,
          children: [...lastlast.children, node(x)]
        }],
      }];
    } else return acc;
  }, []),
  h1: title => ({title: title, level: 0}),
  h2: title => ({title: title, level: 1}),
  h3: title => ({title: title, level: 2}),
  tabs: tabs => [],
  ul: xs => [],
  p: xs => [],
  b: s => [],
  a: (text, href) => [],
  a_external: (text, href) => [],
  code: code => [],
  codeblock: (code, lang) => [],
  icon: () => [],
  table: (hs, xs) => [],
  process_state_diagram: source => [],
};

const renderer_markdown = {
  compose: (xs: string[]) => xs.join("\n"),
  h: (title: string, level: 1|2|3|4) => "#".repeat(level)+" "+title,
  p: (...xs: string[]) => xs.join("")+"\n",
  code: (code: string) => "`"+code+"`",
  codeblock: (lang: string, code: string) => "```"+lang+"\n"+code+"\n```\n",
  a: (text: string, href: string) => `[${text}](${href})`,
  bold: (text: string) => `**${text}**`,
  italic: (text: string) => `_${text}_`,
  ul: (...lines: string[][]) => lines.map(line => renderer_markdown.li(...line)).join("\n")+"\n", // TODO: move out li
  li: (...xs: string[]) => "- "+xs.join(""),
  img: (src: string, alt: string) => `![${alt}](${src})`,
  table: (headers: string[], ...xs: string[][]) => {
    const row = (xs: string[]) => "| "+xs.join(" | ")+" |";
    const header = row(headers);
    const sep = "|"+headers.map(s => ("  "+s).split("").map(() => "-").join("")).join("|",)+"|";
    const rows = xs.map(r => row(r));
    return [header, sep, ...rows].join("\n") + "\n";
  },
  hr: "---",
};

const markdown_adapter: Adapter<string, string> = {
  render: (doc) => renderer_markdown.compose(doc(markdown_adapter)),
  h1: (title: string) => renderer_markdown.h(title, 1),
  h2: (title: string) => renderer_markdown.h(title, 2),
  h3: (title: string) => renderer_markdown.h(title, 3),
  tabs: tabs => tabs.map(([title, ...content]) => {
    const header = renderer_markdown.h(title, 4);
    return header + "\n" + content.join("\n");
  }).join("\n"),
  p: (...xs: string[]) => renderer_markdown.p(...xs),
  b: (s: string) => renderer_markdown.bold(s),
  a: (text: string, href: string) => renderer_markdown.a(text, href), // TODO: local links should work
  a_external: (text: string, href: string) => renderer_markdown.a(text, href),
  code: (code: string) => renderer_markdown.code(code),
  codeblock: (code: string, lang: Lang) => renderer_markdown.codeblock(lang, code),
  ul: (...xs: string[][]) => renderer_markdown.ul(...xs),
  icon: () => '<p align="center"><img src="docs/icon.svg" width="250" height="250"></p>\n',
  table: (headers: string[], ...xs: string[][]) => renderer_markdown.table(headers, ...xs),
  process_state_diagram: source => renderer_markdown.codeblock("mermaid", source),
};

import {icon as github_corner, style as github_corner_style} from "./github-corner.ts";
import css from "./styles.ts";

const diagram = `flowchart TB
  0( )
  S(Stopped)
  C(Created)
  R(Running)
  A{{autorestart/watch enabled?}}
  0 -->|new process| S
  subgraph Running
    direction TB
    C -->|process started| R
    R -->|process died| A
  end
  A -->|yes| C
  A -->|no| S
  Running -->|stop| S
  S -->|start| C`;
const diagurl = "https://kroki.io/mermaid/svg/" + deflate(diagram, {level: 9})
  .toBase64()
  .replace(/\+/g, '-')
  .replace(/\//g, '_');
const diagresponse = await fetch(diagurl);
const diagsvg = await diagresponse.text();

type HTMLNode = string | HTMLElem;
type HTMLElem = readonly [string, Record<string, string | number>, ...HTMLNode[]];
const html_adapter: Adapter<HTMLNode, string> = ((): Adapter<HTMLNode, string> => {
  const join = (
    sep: string,
    fmt: (kv: [k: string, v: string]) => string,
    o: Record<string, string>,
  ) => Object.entries(o).toSorted(([k1, _v1], [k2, _v2]) => k1.localeCompare(k2)).map(fmt).join(sep);
  const renderCSSProps = (o: Record<string, string>) => join(" ", ([k, v]) => `${k}: ${v};`, o);
  const renderCSS = (o: [string, Record<string, string>][]) =>
    o.map(([k, v]) => [k, renderCSSProps(v)]).map(([k, v]) => `${k} { ${v} }`).join("\n");

  const span = (s: string): HTMLNode => ["span", {}, s];
  const li = (xs: HTMLNode[]): HTMLNode => ["li", {}, ...xs];
  const self: Adapter<HTMLNode, string> = {
    render: (doc): string => {
      const TOC: TOCItem2[] = toc.render(doc)[0].children; // NOTE: skip h1
      return "<!DOCTYPE html>"+manifestXmlJsonml(["html", {lang: "en"},
        ["head", {},
          ["title", {}, "pm"],
          ["meta", {"http-equiv": "Content-Type", charset: "UTF-8"}],
          ["meta", {name: "description", content: "process manager"}],
          ["meta", {"http-equiv": "X-UA-Compatible", content: "IE=edge,chrome=1"}],
          ["meta", {name: "viewport", content: "width=device-width, initial-scale=1"}],
          ["meta", {property: "og:title", content: "pm"}],
          ["meta", {property: "og:description", content: "process manager"}],
          ["meta", {property: "og:type", content: "website"}],
          ["meta", {property: "og:url", content: "https://rprtr258.github.io/pm/"}],
          ["meta", {property: "og:image", content: "https://rprtr258.github.io/pm/icon.svg"}],
          ["style", {}, renderCSS(css)],
        ],
        ["body", {class: "sticky", style: renderCSSProps({margin: "0"})},
          ["a", {href: link_github, class: "github-corner", "aria-label": "View source on GitHub"}, github_corner], github_corner_style,
          ["main", {role: "presentation"},
            ["aside", {class: "sidebar", role: "none"},
              ["div", {class: "sidebar-nav", role: "navigation", "aria-label": "primary"}, (() => {
                const a = (id: string, title: string): HTMLElem => ["a", {class: "section-link", href: "#"+id, title: title}, title];
                const toc_render = (xs: TOCItem2[]): HTMLNode => self.ul(...xs.reduce(
                  (acc: HTMLNode[][], x: TOCItem2): HTMLNode[][] => [...acc, [a(x.title, x.title)], [toc_render(x.children)]],
                  [],
                ));
                return toc_render(TOC);
              })()],
            ],
            ["section", {class: "content"},
              ["article", {id: "main", class: "markdown-section", role: "main"}, ...doc(html_adapter)]
            ],
          ],
        ],
      ]);
    },
    process_state_diagram: source => diagsvg, // TODO: solve sync/async
    code: s => {
      const escape = (s: string) => s.split("").map((c) => {
        if (c == "<") return "&lt;";
        else if (c == ">") return "&gt;";
        else return c;}).join("");
      return ["code", {}, escape(s)];
    },
    b: s => ["b", {}, s],
    p: (...xs: HTMLNode[]) => ["p", {}, ...xs],
    ul: (...xs: HTMLNode[][]) => ["ul", {}, ...xs.map(x => li(x))],
    a: (text, href) => ["a", {href: link_github + "/blob/master/" + href}, text],
    a_external: (text, href) => ["a", {href: href, target: "_top"}, text],
    h1: title => ["h1", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    h2: title => ["h2", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    h3: title => ["h3", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    tabs: tabs => {
      return ["p", {},
        ["style", {}, `
          /* Style the tab */
          .tab {
            overflow: hidden;
            background-color: var(--code-theme-background);
          }

          /* Style the buttons that are used to open the tab content */
          .tab button {
            background-color: inherit;
            float: left;
            border: 1px solid transparent;
            outline: none;
            cursor: pointer;
            padding: 8px 14px;
          }

          /* Change background color of buttons on hover */
          .tab button:hover {
            background-color: #ccc8b1;
            border: 1px solid var(--code-theme-background);
          }

          /* Create an active/current tablink class */
          .tab button.active {
            background-color: #ccc8b1;
            border: 1px solid #454138;
          }

          /* Style the tab content */
          .tabcontent {
            display: none;
            padding: 1px 12px;
            background-color: #45413810;
          }`,
        ],
        ["div", {class: "tab"}, ...tabs.map(([title, ..._]): HTMLElem =>
          ["button", {class: "tablinks", onclick: `openCity(event, '${title}')`}, title],
        )],
        // Tab content
        ...tabs.map(([title, ...content]): HTMLElem =>
          ["div", {id: title, class: "tabcontent"},
            ["p", {}, ...content],
          ],
        ),
        ["script", {}, `
          function openCity(evt, cityName) {
            // Get all elements with class="tabcontent" and hide them
            const tabcontent = document.getElementsByClassName("tabcontent");
            for (let i = 0; i < tabcontent.length; i++) {
              tabcontent[i].style.display = "none";
            }

            // Get all elements with class="tablinks" and remove the class "active"
            const tablinks = document.getElementsByClassName("tablinks");
            for (let i = 0; i < tablinks.length; i++) {
              tablinks[i].className = tablinks[i].className.replace(" active", "");
            }

            // Show the current tab, and add an "active" class to the button that opened the tab
            document.getElementById(cityName).style.display = "block";
            evt.currentTarget.className += " active";
          }
          document.querySelector(".tab > button:nth-child(1)").click();
        `],
      ];
    },
    icon: () => ["p", {align: "center"}, ["img", {src: "icon.svg", width: 250, height: 250, style: renderCSSProps({border: "0"})}]],
    codeblock: (code, lang) => {
      if (lang === "sh") {
        const functionn = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-function)"})}, s];
        const variable = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-variable)"})}, s];
        const comment = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-comment)"})}, s];
        const operator = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-operator)"})}, s];
        const env = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-tag)"})}, s];
        const number = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-tag)"})}, s];
        const punctuation = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-punctuation)"})}, s];
        const render_word = (word: string): HTMLNode[] => {
          const functionns = ["wget", "chmod", "chown", "mv", "cp", "ln", "sudo", "git"];
          const lt = std.findSubstr("<", word);
          const gt = std.findSubstr(">", word);
          const lsb = std.findSubstr("[", word);
          const rsb = std.findSubstr("]", word);
          const ellipsis = std.findSubstr("...", word);
          if (word == "") return [];
          else if (lt.length  > 0) return [...render_word(std.substr(word, 0, lt[0])),  operator("&lt;"), ...render_word(std.substr(word,  lt[0]+1, word.length))];
          else if (gt.length  > 0) return [...render_word(std.substr(word, 0, gt[0])),  operator("&gt;"), ...render_word(std.substr(word,  gt[0]+1, word.length))];
          else if (lsb.length > 0) return [
            ...render_word(word.substring(0, lsb[0])),
            punctuation("["),
            ...render_word(word.substring(lsb[0]+1, word.length)),
          ];
          else if (rsb.length > 0) return [...render_word(std.substr(word, 0, rsb[0])), punctuation("]"), ...render_word(std.substr(word, rsb[0]+1, word.length))];
          else if (ellipsis.length > 0) return [...render_word(std.substr(word, 0, ellipsis[0])), punctuation("..."), ...render_word(std.substr(word, ellipsis[0]+3, word.length))];
          else if (functionns.some(x => word == x)) return [functionn(word)];
          else if (word == "[") return [punctuation(word)];
          else if (word == "]") return [punctuation(word)];
          else if (word == "644") return [number(word)];
          else if (word == "enable") return [["span", {class: "token class-name", style: renderCSSProps({color: "var(--code-theme-selector)"})}, "enable"]];
          else if (word[0] == "-") return [variable(word)];
          else if (word == "$HOME/.pm/") return [env("$HOME"), "/.pm/"];
          else return [word];
        };
        const render = (line: string): HTMLNode[] => { // TODO: use sh parser actually
          if (line === "")
            return [];
          else if (line.indexOf("#") !== -1) {
            const hash = line.indexOf("#");
            const before = line.substring(0, hash);
            const after = line.substring(hash, line.length);
            return [...render(before), comment(after)];
          } else return std.joinList([" "], line.split(" ").map(render_word));
        };
        const lines = code.split("\n").map(render);
        return ["pre", {"data-lang": lang, style: renderCSSProps({background: "var(--code-theme-background)"})},
          ["code", {class: "language-" + lang}, ...std.joinList(["\n"], lines), "\n"]];
      }

      const punctuation = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-tag)"})}, s];
      const escape = (s: string) => s.split("").map(c => {
        if (c == "<") return "&lt;";
        else if (c == ">") return "&gt;";
        else if (c == "[" || c == "]" || c == "{" || c == "}" || c == "=") return punctuation(c);
        else return c;});
      return ["pre", {"data-lang": lang, style: renderCSSProps({background: "var(--code-theme-background)"})},
        ["code", {class: "language-" + lang}, ...escape(code)]];
    },
    table: (headers, ...xs) => ["table", {},
      ["thead", {},
        ["tr", {}, ...headers.map((x): HTMLElem => ["th", {}, x])],
      ],
      ["tbody", {}, ...xs.map((r): HTMLElem =>
        ["tr", {}, ...r.map((x): HTMLElem => ["td", {}, x])]
      )],
    ],
  };
  return self;
})();

const link_github = "https://github.com/rprtr258/pm";
const link_release = link_github + "/releases/latest";

const docs = <T, X>({
  h1, h2, h3, tabs,
  p, b, code, a, a_external,
  codeblock, ul, table,
  icon, process_state_diagram,
}: Adapter<T, X>): T[] => [
  h1("PM (process manager)"),
    icon(),
    h2("Installation"),
      p("PM is available only for linux due to heavy usage of linux mechanisms. Go to the ", a_external("releases", link_release), " page to download the latest binary."),
      codeblock(dedent(`
        # download binary
        wget ${link_release}/download/pm_linux_amd64
        # make binary executable
        chmod +x pm_linux_amd64
        # move binary to $PATH, here just local
        mv pm_linux_amd64 pm
      `), "sh"),
      h3("Systemd service"),
        p("To enable running processes on system startup:"),
        codeblock(dedent(`
          # soft link /usr/bin/pm binary to whenever it is installed
          sudo ln -s ~/go/bin/pm /usr/bin/pm
          # install systemd service, copy/paste output of following command
          pm startup
        `), "sh"),
        p("After these commands, processes with ", code("startup: true"), " config option will be started on system startup."),
    h2("Configuration"),
      p("PM supports multiple configuration formats for defining processes. The original ", a_external("jsonnet", "https://jsonnet.org/"), " format is supported, along with several additional formats for flexibility:"),
      h3("Supported Formats"), tabs([
        ["JSONNet (.jsonnet)",
          p("The primary configuration format. JSONNet is fully compatible with plain JSON."),
          codeblock(configExamples.jsonnet, "jsonnet")],
        ["YAML (.yaml, .yml)",
          p(a_external("YAML", "https://yaml.org/"), a_external(" ", "https://noyaml.com/"), "- Human-readable data serialization standard."),
          codeblock(configExamples.yaml, "yaml")],
        ["TOML (.toml)",
          p(a_external("TOML", "https://toml.io/"), " - Tom's Obvious, Minimal Language configuration format."),
          codeblock(configExamples.toml, "toml")],
        ["INI (.ini, .cfg, .conf)",
          p("Classic configuration file format with section-based structure."),
          codeblock(configExamples.ini, "ini")],
        ["HCL (.hcl)",
          p("HashiCorp Configuration Language, designed for human-readable machine-friendly configs."),
          codeblock(configExamples.hcl, "hcl")],
        ["JSON (.json)",
          p("Plain JSON configuration format."),
          codeblock(configExamples.json, "json")],
      ]),
      h3("Configuration Schema"),
        p("All formats define list of processes with following fields:"),
        table(
          ["Field",              "Type",           "Description",                              "Required"],
          [code("name"),         code("string"),   "Process name",                              "Yes"],
          [code("command"),      code("string"),   "Command to execute",                       "Yes"],
          [code("args"),         code("array(string)"), "Command arguments",                   "No"],
          [code("cwd"),          code("string"),   "Working directory",                        "No"],
          [code("env"),          code("map(string, string)"),   "Environment variables (name: value pairs)", "No"],
          [code("tags"),         code("array(string)"), "Process tags for filtering",            "No"],
          [code("watch"),        code("string"),   "File pattern to watch for restarts (regex)", "No"],
          [code("startup"),      code("boolean"),  "Start process on system startup",            "No"],
          [code("depends_on"),   code("array(string)"), "Process names that must start first",   "No"],
          [code("cron"),         code("string"),   "Cron expression for scheduled execution",    "No"],
          [code("stdout_file"),  code("string"),   "File to redirect stdout to",                 "No"],
          [code("stderr_file"),  code("string"),   "File to redirect stderr to",                 "No"],
          [code("kill_timeout"), code("duration"), "Time before SIGKILL after SIGINT",           "No"],
          [code("autorestart"),  code("boolean"),  "Auto-restart on process death",              "No"],
          [code("max_restarts"), code("number"),   "Maximum restart limit (0 = unlimited)",      "No"],
        ),
        p("See ", a("example configuration file", "./config.jsonnet"), ". Other examples can be found in ", a("tests", "./e2e/tests"), " directory."),
    h2("Usage"),
      p("Most fresh usage descriptions can be seen using ", code("pm <command> --help"), "."),
      h3("Run process"),
        codeblock(dedent(`
          # run process using command
          pm run go run main.go

          # run processes from config file
          pm run --config config.jsonnet
        `), "sh"),
      h3("List processes"),
        codeblock(dedent(`
          pm list
        `), "sh"),
      h3("Start already added processes"),
        codeblock(dedent(`
          pm start [ID/NAME/TAG]...
        `), "sh"),
      h3("Stop processes"),
        codeblock(dedent(`
          pm stop [ID/NAME/TAG]...

          # e.g. stop all added processes (all processes has tag `+"`all`"+` by default)
          pm stop all
        `), "sh"),
      h3("Delete processes"),
        p("When deleting process, they are first stopped, then removed from ", code("pm"), "."),
        codeblock(dedent(`
          pm delete [ID/NAME/TAG]...

          # e.g. delete all processes
          pm delete all
        `), "sh"),
    h2("Process state diagram"),
      process_state_diagram(diagram),
    h2("Development"),
      h3("Architecture"),
        p(code("pm"), " consists of two parts:"),
        ul(
          [b("cli client"), " - requests server, launches/stops shim processes"],
          [b("shim"), " - monitors and restarts processes, handle watches, signals and shutdowns"],
        ),
      h3("PM directory structure"),
        p(
          code("pm"),
          " uses ",
          a_external("XDG", "https://specifications.freedesktop.org/basedir-spec/latest/"),
          " specification, so db and logs are in ",
          code("~/.local/share/pm"),
          " and config is ",
          code("~/.config/pm.json"),
          ". ",
          code("XDG_DATA_HOME"), " and ", code("XDG_CONFIG_HOME"),
          " environment variables can be used to change this. Layout is following:"),
          codeblock(dedent(`
            ~/.config/pm.json # pm config file
            ~/.local/share/pm/
            ├──db/ # database tables
            │   └──<ID> # process info
            └──logs/ # processes logs
                ├──<ID>.stdout # stdout of process with id ID
                └──<ID>.stderr # stderr of process with id ID
          `), "sh"),
      h3("Differences from pm2"),
        ul(
          [code("pm"), " is just a single binary, not dependent on ", code("nodejs"), " and bunch of ", code("js"), " scripts"],
          [a_external("jsonnet", "https://jsonnet.org/"), " configuration language, back compatible with ", code("JSON"), " and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in ", code("pm"), " (others configuration languages might be added in future such as ", code("Procfile"), ", ", code("HCL"), ", etc.)"],
          ["supports only ", code("linux"), " now"],
          ["I can fix problems/add features as I need, independent of whether they work or not in ", code("pm2"), " because I don't know ", code("js")],
          ["fast and convenient (I hope so)"],
          ["no specific integrations for ", code("js")],
        ),
      h3("Release"),
        p("On ", code("master"), " branch:"),
        codeblock(dedent(`
          git tag v1.2.3
          git push --tags
          GITHUB_TOKEN=<token> goreleaser release --clean
        `), "sh"),
];

type Link = {
  link: string,
  isExternal: boolean,
};

const links_collect: Adapter<Link[], Link[]> = {
  render: (doc) => doc(links_collect).flatMap(ss => ss),
  h1: title => [],
  h2: title => [],
  h3: title => [],
  tabs: title => [],
  ul: (...xs) => xs.flatMap(ss => ss.flatMap(s => typeof s === "string" ? [] : s)),
  p: (...xs) => xs.flatMap(s => typeof s === "string" ? [] : s),
  b: s => [],
  a: (text, link) => [{link, isExternal: false}],
  a_external: (text, link) => [{link, isExternal: true}],
  code: code => [],
  codeblock: (code, lang) => [],
  icon: () => [],
  table: (...xs) => xs.flatMap(ss => ss.flatMap(s => typeof s === "string" ? [] : s)),
  process_state_diagram: source => [],
};

async function writeFile(filename: string, content: string): Promise<void> {
  const dir = import.meta.dir;
  console.log("Writing", filename + "...");
  await Bun.write(dir + "/" + filename, content);
}

const workspaceDir = join(import.meta.dir, "..");
// Read example config files for documentation
const configExamples = {
  jsonnet: await Bun.file(import.meta.dir + "/examples/config.jsonnet").text(),
  yaml: await Bun.file(import.meta.dir + "/examples/config.yaml").text(),
  toml: await Bun.file(import.meta.dir + "/examples/config.toml").text(),
  ini: await Bun.file(import.meta.dir + "/examples/config.ini").text(),
  hcl: await Bun.file(import.meta.dir + "/examples/config.hcl").text(),
  json: await Bun.file(import.meta.dir + "/examples/config.json").text(),
};

console.log("Checking links...");
await Promise.all([...new Set(links_collect.render(docs))].map(async ({link, isExternal}) => {
  if (isExternal) {
    const res = await fetch(link);
    if (res.status < 200 || res.status >= 300)
      throw new Error(`Broken link: ${link}`);
    console.log(styleText("greenBright", "OK") + ":", styleText(["underline", "blueBright"], link), "=>", styleText("greenBright", `${res.status}`));
  } else {
    const info = await stat(join(workspaceDir, link));
    const type = info.isDirectory() ? "DIR" :
      info.isFile() ? "FILE" :
      (() => {throw new Error(`Broken local name: ${link}`)})();
    console.log(styleText("greenBright", "OK") + ":", link, "=>", styleText("blue", type));
  }
}))
console.log("All links OK!");
console.log();

await writeFile("index.html", html_adapter.render(docs));
await writeFile("../readme.md", markdown_adapter.render(docs));
