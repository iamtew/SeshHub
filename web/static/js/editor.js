(function () {
  var ta = document.querySelector("textarea[name=content_raw]");
  if (!ta) return;
  var s = document.createElement("script");
  s.src = "/static/monaco/vs/loader.js";
  s.onload = function () {
    window.require.config({ paths: { vs: "/static/monaco/vs" } });
    window.require(["vs/editor/editor.main"], function () {
      var mount = document.createElement("div");
      mount.id = "monaco-mount";
      ta.insertAdjacentElement("afterend", mount);
      ta.classList.add("monaco-src");
      var ed = window.monaco.editor.create(mount, {
        value: ta.value,
        language: "markdown",
        theme: "vs-dark",
        automaticLayout: true,
        minimap: { enabled: false }
      });
      if (ta.form) ta.form.addEventListener("submit", function () { ta.value = ed.getValue(); });
    });
  };
  document.head.appendChild(s);
})();
