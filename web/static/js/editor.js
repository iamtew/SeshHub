(function () {
  var wrap = document.querySelector(".md-edit");
  var ta = wrap && wrap.querySelector("textarea");
  if (!ta) return;
  var basicPrev = wrap.querySelector(".md-preview");
  var fill = wrap.getAttribute("data-fill");
  var form = ta.form;
  var meta = document.querySelector("[data-md-meta]");
  var dialog = document.querySelector("dialog.md-overlay");
  var side = dialog && dialog.querySelector(".md-side");
  var advPrev = dialog && dialog.querySelector(".md-overlay-preview");
  var mount = dialog && dialog.querySelector("#monaco-mount");
  var t = 0;
  var ed;
  var loading;
  var metaHome;
  var metaNext;

  var CDN = "https://cdn.jsdelivr.net/npm/monaco-editor@0.52.2/min";

  function paint(html) {
    if (basicPrev) basicPrev.innerHTML = html;
    if (advPrev) advPrev.innerHTML = html;
    if (typeof window.mdDates === "function") window.mdDates();
  }

  function preview() {
    var raw = ed ? ed.getValue() : ta.value;
    var body = "content_raw=" + encodeURIComponent(raw);
    if (fill) body += "&fill=" + encodeURIComponent(fill);
    fetch("/preview", {
      method: "POST",
      credentials: "same-origin",
      headers: { "Content-Type": "application/x-www-form-urlencoded" },
      body: body
    }).then(function (r) {
      if (!r.ok) return "";
      return r.text();
    }).then(paint).catch(function () {});
  }

  function debounce() {
    clearTimeout(t);
    t = setTimeout(preview, 300);
  }

  ta.addEventListener("input", debounce);
  preview();

  function insertText(text) {
    if (ed && dialog && dialog.open) {
      ed.executeEdits("insert", [{ range: ed.getSelection(), text: text, forceMoveMarkers: true }]);
      ed.focus();
      debounce();
      return;
    }
    var s = ta.selectionStart;
    var e = ta.selectionEnd;
    ta.value = ta.value.slice(0, s) + text + ta.value.slice(e);
    ta.selectionStart = ta.selectionEnd = s + text.length;
    ta.focus();
    debounce();
  }

  document.addEventListener("click", function (ev) {
    var btn = ev.target.closest(".md-insert");
    if (!btn) return;
    ev.preventDefault();
    insertText(btn.getAttribute("data-insert") || "");
  });

  function bindForm(root, id) {
    if (!root || !id) return;
    var nodes = root.querySelectorAll("input, select, textarea");
    for (var i = 0; i < nodes.length; i++) nodes[i].setAttribute("form", id);
  }

  function theme(monaco) {
    monaco.editor.defineTheme("sesh-sofa", {
      base: "vs-dark",
      inherit: true,
      rules: [
        { token: "", foreground: "E0E0E0" },
        { token: "comment", foreground: "9AAC62" },
        { token: "keyword", foreground: "8600DF" },
        { token: "string", foreground: "E5F20D" },
        { token: "number", foreground: "00FFFF" },
        { token: "regexp", foreground: "00FFFF" },
        { token: "type", foreground: "00F53A" },
        { token: "class", foreground: "00F53A" },
        { token: "function", foreground: "00F53A" },
        { token: "variable", foreground: "E0E0E0" },
        { token: "tag", foreground: "FF00FF" },
        { token: "attribute.name", foreground: "00FFFF" },
        { token: "attribute.value", foreground: "E5F20D" },
        { token: "emphasis", fontStyle: "italic", foreground: "E5F20D" },
        { token: "strong", fontStyle: "bold", foreground: "E5F20D" },
        { token: "keyword.md", foreground: "8600DF" },
        { token: "string.md", foreground: "E5F20D" },
        { token: "variable.md", foreground: "00FFFF" },
        { token: "tag.md", foreground: "FF00FF" }
      ],
      colors: {
        "editor.background": "#000000",
        "editor.foreground": "#E0E0E0",
        "editorLineNumber.foreground": "#6C754A",
        "editorLineNumber.activeForeground": "#9AAC62",
        "editorCursor.foreground": "#00F53A",
        "editor.selectionBackground": "#8600DF66",
        "editor.inactiveSelectionBackground": "#8600DF33",
        "editor.lineHighlightBackground": "#6C754A33",
        "editorIndentGuide.background": "#6C754A",
        "editorIndentGuide.activeBackground": "#9AAC62",
        "editorWhitespace.foreground": "#666666",
        "editorError.foreground": "#F20D0D",
        "editorWarning.foreground": "#E5F20D",
        "editorInfo.foreground": "#00FFFF",
        "editorGutter.background": "#000000",
        "editorWidget.background": "#000000",
        "editorWidget.border": "#8600DF",
        "input.background": "#000000",
        "input.foreground": "#E0E0E0",
        "focusBorder": "#00F53A",
        "scrollbarSlider.background": "#6C754A66",
        "scrollbarSlider.hoverBackground": "#9AAC62aa"
      }
    });
  }

  function openAdv() {
    dialog = document.querySelector("dialog.md-overlay");
    side = dialog && dialog.querySelector(".md-side");
    advPrev = dialog && dialog.querySelector(".md-overlay-preview");
    mount = dialog && dialog.querySelector("#monaco-mount");
    form = ta.form;
    if (!dialog || !form) return;
    if (!form.id) form.id = "md-form";
    if (meta && side && meta.parentNode !== side) {
      metaHome = meta.parentNode;
      metaNext = meta.nextSibling;
      bindForm(meta, form.id);
      side.appendChild(meta);
    }
    if (typeof dialog.showModal === "function") dialog.showModal();
    else dialog.setAttribute("open", "");
    loadMonaco(function () {
      if (ed) {
        ed.setValue(ta.value);
        ed.layout();
        ed.focus();
        debounce();
        return;
      }
      theme(window.monaco);
      ed = window.monaco.editor.create(mount, {
        value: ta.value,
        language: "markdown",
        theme: "sesh-sofa",
        automaticLayout: true,
        minimap: { enabled: false },
        wordWrap: "on",
        fontSize: 15,
        padding: { top: 8 }
      });
      ed.onDidChangeModelContent(debounce);
      debounce();
    });
  }

  function closeAdv() {
    if (ed) ta.value = ed.getValue();
    if (meta && metaHome) {
      metaHome.insertBefore(meta, metaNext);
    }
    debounce();
  }

  function loadMonaco(done) {
    if (window.monaco) {
      done();
      return;
    }
    if (loading) {
      loading.push(done);
      return;
    }
    loading = [done];
    window.MonacoEnvironment = {
      getWorkerUrl: function () {
        if (!window._seshMonacoWorker) {
          window._seshMonacoWorker = URL.createObjectURL(new Blob([
            "self.MonacoEnvironment={baseUrl:'" + CDN + "/'};",
            "importScripts('" + CDN + "/vs/base/worker/workerMain.js');"
          ], { type: "text/javascript" }));
        }
        return window._seshMonacoWorker;
      }
    };
    var s = document.createElement("script");
    s.src = CDN + "/vs/loader.js";
    s.onload = function () {
      window.require.config({ paths: { vs: CDN + "/vs" } });
      window.require(["vs/editor/editor.main"], function () {
        var cbs = loading;
        loading = null;
        for (var i = 0; i < cbs.length; i++) cbs[i]();
      });
    };
    s.onerror = function () {
      loading = null;
      if (mount) mount.textContent = "Monaco CDN failed. Close and use the basic editor.";
    };
    document.head.appendChild(s);
  }

  document.addEventListener("click", function (ev) {
    if (ev.target.closest(".md-adv-btn")) {
      ev.preventDefault();
      openAdv();
    }
  });
  if (dialog) {
    dialog.querySelector(".md-overlay-close").addEventListener("click", function () {
      dialog.close();
    });
    dialog.querySelector(".md-overlay-save").addEventListener("click", function () {
      if (ed) ta.value = ed.getValue();
      dialog.close();
      var stay = form && form.querySelector('[name=after][value=stay]');
      if (stay) stay.click();
      else if (form) form.requestSubmit();
    });
    dialog.addEventListener("close", closeAdv);
  }
  if (form) {
    form.addEventListener("submit", function () {
      if (ed) ta.value = ed.getValue();
    });
  }
})();
