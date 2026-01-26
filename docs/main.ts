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
    // .join(tag === "span" ? " " : "");
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

type Adapter<T, X> = {
  render: (doc: <R, Y>(a: Adapter<R, Y>) => R[]) => X,
  h1: (title: string) => T,
  h2: (title: string) => T,
  h3: (title: string) => T,
  h4: (title: string) => T,
  p: (xs: (string | T)[]) => T,
  b: (s: string) => T,
  a: (text: string, href: string) => T,
  a_external: (text: string, href: string) => T,
  code: (code: string) => T,
  codeblock_sh: (code: string) => T,
  codeblock_jsonnet: (code: string) => T,
  codeblock_yaml: (code: string) => T,
  codeblock_toml: (code: string) => T,
  codeblock_ini: (code: string) => T,
  codeblock_hcl: (code: string) => T,
  codeblock_json: (code: string) => T,
  codeblock: (code: string) => T,
  ul: (xs: (string | T)[][]) => T,
  icon: () => T,
  process_state_diagram: T,
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
  h1: (title: string) => ({title: title, level: 0}),
  h2: (title: string) => ({title: title, level: 1}),
  h3: (title: string) => ({title: title, level: 2}),
  h4: (title: string) => ({title: title, level: 3}),
  ul: (xs) => [],
  p: (xs) => [],
  b: (s) => [],
  a: (text, href) => [],
  a_external: (text, href) => [],
  code: (code) => [],
  codeblock_sh: (code) => [],
  codeblock_jsonnet: (code) => [],
  codeblock_yaml: (code) => [],
  codeblock_toml: (code) => [],
  codeblock_ini: (code) => [],
  codeblock_hcl: (code) => [],
  codeblock_json: (code) => [],
  codeblock: (code) => [],
  icon: () => [],
  process_state_diagram: [],
};

const renderer_markdown = {
  compose: (xs: string[]) => xs.join("\n"),
  h1: (title: string) => "# "+title,
  h2: (title: string) => "## "+title,
  h3: (title: string) => "### "+title,
  h4: (title: string) => "#### "+title,
  p: (xs: string[]) => xs.join("")+"\n",
  code: (code: string) => "`"+code+"`",
  codeblock: (lang: string, code: string) => "```"+lang+"\n"+code+"\n```\n",
  a: (text: string, href: string) => `[${text}](${href})`,
  bold: (text: string) => `**${text}**`,
  italic: (text: string) => `_${text}_`,
  ul: (lines: string[][]) => lines.map(line => renderer_markdown.li(line)).join("\n")+"\n", // TODO: move out li
  li: (x: string[]) => "- "+x.join(""),
  img: (src: string, alt: string) => `![${alt}](${src})`,
  hr: "---",
};

const markdown_adapter: Adapter<string, string> = {
  render: (doc) => renderer_markdown.compose(doc(markdown_adapter)),
  h1: (title: string) => renderer_markdown.h1(title),
  h2: (title: string) => renderer_markdown.h2(title),
  h3: (title: string) => renderer_markdown.h3(title),
  h4: (title: string) => renderer_markdown.h4(title),
  p: (xs: string[]) => renderer_markdown.p(xs),
  b: (s: string) => renderer_markdown.bold(s),
  a: (text: string, href: string) => renderer_markdown.a(text, href), // TODO: local links should work
  a_external: (text: string, href: string) => renderer_markdown.a(text, href),
  code: (code: string) => renderer_markdown.code(code),
  codeblock_sh: (code: string) => renderer_markdown.codeblock("sh", code),
  codeblock_jsonnet: (code: string) => renderer_markdown.codeblock("jsonnet", code),
  codeblock_yaml: (code: string) => renderer_markdown.codeblock("yaml", code),
  codeblock_toml: (code: string) => renderer_markdown.codeblock("toml", code),
  codeblock_ini: (code: string) => renderer_markdown.codeblock("ini", code),
  codeblock_hcl: (code: string) => renderer_markdown.codeblock("hcl", code),
  codeblock_json: (code: string) => renderer_markdown.codeblock("json", code),
  codeblock: (code: string) => renderer_markdown.codeblock("", code),
  ul: (xs: string[][]) => renderer_markdown.ul(xs),
  icon: () => '<p align="center"><img src="docs/icon.svg" width="250" height="250"></p>\n',
  process_state_diagram: renderer_markdown.codeblock("mermaid", dedent(`
    flowchart TB
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
      Running  -->|stop| S
      S -->|start| C
  `)),
};

import process_state_diagram from "./process-state-diagram.ts"; // TODO: render from mermaid
import css from "./styles.ts";

type HTMLNode = string | HTMLElem;
type HTMLElem = [string, Record<string, string | number>, ...HTMLNode[]];
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
  const img = (src: string, width: number, height: number): HTMLNode => ["img", {src: src, width: width, height: height, style: renderCSSProps({border: "0"})}];
  const codeblock_generic = (lang: string, code: string): HTMLNode => {
    const punctuation = (s: string): HTMLElem => ["span", {style: renderCSSProps({color: "var(--code-theme-tag)"})}, s];
    const escape = (s: string) => s.split("").map(c => {
      if (c == "<") return "&lt;";
      else if (c == ">") return "&gt;";
      else if (c == "[" || c == "]" || c == "{" || c == "}" || c == "=") return punctuation(c);
      else return c;});
    return ["pre", {"data-lang": lang, style: renderCSSProps({background: "var(--code-theme-background)"})},
      ["code", {class: "language-" + lang}, ...escape(code)]];
  };
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
          ["main", {role: "presentation"},
            ["aside", {class: "sidebar", role: "none"},
              ["div", {class: "sidebar-nav", role: "navigation", "aria-label": "primary"}, (() => {
                const a = (id: string, title: string): HTMLElem => ["a", {class: "section-link", href: "#"+id, title: title}, title];
                const toc_render = (xs: TOCItem2[]): HTMLNode => self.ul(xs.reduce(
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
    process_state_diagram: process_state_diagram as HTMLElem,
    code: (s) => {
      const escape = (s: string) => s.split("").map((c) => {
        if (c == "<") return "&lt;";
        else if (c == ">") return "&gt;";
        else return c;}).join("");
      return ["code", {}, escape(s)];
    },
    b: (s) => ["b", {}, s],
    p: (xs: HTMLNode[]) => ["p", {}, ...xs],
    ul: (xs: HTMLNode[][]) => ["ul", {}, ...xs.map(x => li(x))],
    a: (text, href) => ["a", {href: "https://github.com/rprtr258/pm/blob/master/" + href}, text],
    a_external: (text, href) => ["a", {href: href, target: "_top"}, text],
    h1: (title) => ["h1", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    h2: (title) => ["h2", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    h3: (title) => ["h3", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    h4: (title) => ["h4", {id: title}, ["a", {href: "#"+title, class: "anchor"}, span(title)]],
    icon: () => ["p", {align: "center"}, ["img", {src: "icon.svg", width: 250, height: 250, style: renderCSSProps({border: "0"})}]],
    codeblock_sh: (code) => {
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
      return ["pre", {"data-lang": "sh", style: renderCSSProps({background: "var(--code-theme-background)"})},
        ["code", {class: "language-sh"}, ...std.joinList(["\n"], lines), "\n"]];
    },
    codeblock_jsonnet: (code) => codeblock_generic("jsonnet", code),
    codeblock_yaml: (code) => codeblock_generic("yaml", code),
    codeblock_toml: (code) => codeblock_generic("toml", code),
    codeblock_ini: (code) => codeblock_generic("ini", code),
    codeblock_hcl: (code) => codeblock_generic("hcl", code),
    codeblock_json: (code) => codeblock_generic("json", code),
    codeblock: (code) => codeblock_generic("", code),
  };
  return self;
})();

const link_release = "https://github.com/rprtr258/pm/releases/latest";

const docs = <T, X>(R: Adapter<T, X>): T[] => [
  R.h1("PM (process manager)"),

  // ["div", {}, R.a("https://github.com/rprtr258/pm", R.img("https://img.shields.io/badge/source-code?logo=github&label=github"))],
  R.icon(),
  R.h2("Installation"),
  R.p(["PM is available only for linux due to heavy usage of linux mechanisms. Go to the ", R.a_external("releases", link_release), " page to download the latest binary."]),
  R.codeblock_sh(dedent(`
    # download binary
    wget ${link_release}/download/pm_linux_amd64
    # make binary executable
    chmod +x pm_linux_amd64
    # move binary to $PATH, here just local
    mv pm_linux_amd64 pm
  `)),
    R.h3("Systemd service"),
    R.p(["To enable running processes on system startup:"]),
    R.codeblock_sh(dedent(`
      # soft link /usr/bin/pm binary to whenever it is installed
      sudo ln -s ~/go/bin/pm /usr/bin/pm
      # install systemd service, copy/paste output of following command
      pm startup
    `)),
    R.p(["After these commands, processes with ", R.code("startup: true"), " config option will be started on system startup."]),

  R.h2("Configuration"),
  R.p([R.a_external("jsonnet", "https://jsonnet.org/"), " configuration language is used. It is also fully compatible with plain JSON, so you can write JSON instead."]),

  R.p(["See ", R.a("example configuration file", "./config.jsonnet"), ". Other examples can be found in ", R.a("tests", "./e2e/tests"), " directory."]),

  R.h2("Usage"),
  R.p(["Most fresh usage descriptions can be seen using ", R.code("pm <command> --help"), "."]),
    R.h3("Run process"),
    R.codeblock_sh(dedent(`
      # run process using command
      pm run go run main.go

      # run processes from config file
      pm run --config config.jsonnet
    `)),
    R.h3("List processes"),
    R.codeblock_sh(dedent(`
      pm list
    `)),

    R.h3("Start already added processes"),
    R.codeblock_sh(dedent(`
      pm start [ID/NAME/TAG]...
    `)),

    R.h3("Stop processes"),
    R.codeblock_sh(dedent(`
      pm stop [ID/NAME/TAG]...

      # e.g. stop all added processes (all processes has tag `+"`all`"+` by default)
      pm stop all
    `)),
    R.h3("Delete processes"),
    R.p(["When deleting process, they are first stopped, then removed from ", R.code("pm"), "."]),
    R.codeblock_sh(dedent(`
      pm delete [ID/NAME/TAG]...

      # e.g. delete all processes
      pm delete all
    `)),

  R.h2("Process state diagram"),
  R.process_state_diagram,

  R.h2("Development"),
    R.h3("Architecture"),
    R.p([R.code("pm"), " consists of two parts:"]),
    R.ul([
      [R.b("cli client"), " - requests server, launches/stops shim processes"],
      [R.b("shim"), " - monitors and restarts processes, handle watches, signals and shutdowns"],
    ]),

    R.h3("PM directory structure"),
    R.p([
      R.code("pm"),
      " uses ",
      R.a("XDG", "https://specifications.freedesktop.org/basedir-spec/latest/"),
      " specification, so db and logs are in ",
      R.code("~/.local/share/pm"),
      " and config is ",
      R.code("~/.config/pm.json"),
      ". ",
      R.code("XDG_DATA_HOME"), " and ", R.code("XDG_CONFIG_HOME"),
      " environment variables can be used to change this. Layout is following:"]),
    R.codeblock_sh(dedent(`
      ~/.config/pm.json # pm config file
      ~/.local/share/pm/
      ├──db/ # database tables
      │   └──<ID> # process info
      └──logs/ # processes logs
          ├──<ID>.stdout # stdout of process with id ID
          └──<ID>.stderr # stderr of process with id ID
    `)),

    R.h3("Differences from pm2"),
    R.ul([
      [R.code("pm"), " is just a single binary, not dependent on ", R.code("nodejs"), " and bunch of ", R.code("js"), " scripts"],
      [R.a_external("jsonnet", "https://jsonnet.org/"), " configuration language, back compatible with ", R.code("JSON"), " and allows to thoroughly configure processes, e.g. separate environments without requiring corresponding mechanism in ", R.code("pm"), " (others configuration languages might be added in future such as ", R.code("Procfile"), ", ", R.code("HCL"), ", etc.)"],
      ["supports only ", R.code("linux"), " now"],
      ["I can fix problems/add features as I need, independent of whether they work or not in ", R.code("pm2"), " because I don't know ", R.code("js")],
      ["fast and convenient (I hope so)"],
      ["no specific integrations for ", R.code("js")],
    ]),

    R.h3("Release"),
    R.p(["On ", R.code("master"), " branch:"]),
    R.codeblock_sh(dedent(`
      git tag v1.2.3
      git push --tags
      GITHUB_TOKEN=<token> goreleaser release --clean
    `)),
];

async function writeFile(filename: string, content: string): Promise<void> {
    const dir = import.meta.dir;
    console.log(filename);
    await Bun.write(dir + "/" + filename, content);
}

await writeFile("index.html", html_adapter.render(docs));
await writeFile("../readme.md", markdown_adapter.render(docs));
