// utils
local items(o) = [{k: k, v: o[k]} for k in std.objectFields(o)];
local join(sep, fmt, o) = std.join(sep, [fmt % kv for kv in items(o)]);
local renderCSSProps(o) = join(" ", "%(k)s: %(v)s;", o);
local renderCSS(o) = join(
  "\n", "%(k)s { %(v)s }",
  std.mapWithKey(function(_, props) renderCSSProps(props), o),
);


// config
local repo = "rprtr258/pm";
local title = repo;
local description = "%(title)s is cool!" % {title: title};
local links = [
  // {"text": "Link 1", "link": "/link"},
];
// available themes as to https://docsify.js.org/#/themes
// vue, buble, dark, pure
local theme = "//cdn.jsdelivr.net/npm/docsify@4/lib/themes/%(theme)s.css" % {theme: "buble"};
local config = {
  subMaxLevel: 1,
  maxLevel: 3,
  auto2top: true,
  repo: repo,
  routerMode: 'history',
  homepage: "readme.md",
};
local style = renderCSS({
  ".markdown-section": { "max-width": "90%" },
  "body": { "font-size": "13pt" },
});

// docs
local dom = ["html", {lang: "en"},
  ["head", {},
    ["meta", {charset: "UTF-8"}],
    ["title", {}, title],
    ["meta", {"http-equiv": "X-UA-Compatible", content: "IE=edge,chrome=1"}],
    ["meta", {name: "description", content: description}],
    ["meta", {name: "viewport", content: "width=device-width, user-scalable=no, initial-scale=1.0, maximum-scale=1.0, minimum-scale=1.0"}],
    ["link", {rel: "stylesheet", href: theme}],
    ["style", {}, style],
  ],
  ["body", {},
    ["nav", {style: renderCSSProps({
      display: "flex",
      "flex-direction": "row",
      "align-items": "center",
      "justify-content": "space-between",
    })},
      ["ul", {}] + [["li", {href: link.link}, link.text] for link in links],
    ],
    ["div", {id: "app"}],
    ["script", {}, |||
      window.$docsify = %(config)s;
    ||| % {config: std.manifestJson(config)}],
    ["script", {src: "//cdn.jsdelivr.net/npm/docsify@4.12.2/lib/docsify.min.js"}],
    ["script", {src: "//cdn.jsdelivr.net/npm/prismjs@1.28.0/prism.min.js"}],
    ["script", {src: "//unpkg.com/docsify/lib/plugins/search.min.js"}],
    ["script", {type: "module"}, |||
      import mermaid from "https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.esm.min.mjs";
      mermaid.initialize({ startOnLoad: true });
      window.mermaid = mermaid;
    |||],
    ["script", {src: "//unpkg.com/docsify-mermaid@2.0.1/dist/docsify-mermaid.js"}],
  ],
];

{
  "index.html": "<!DOCTYPE html>"+std.manifestXmlJsonml(dom),
  "readme.md": importstr "../readme.md",
}
