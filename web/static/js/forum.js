(function () {
  var TW = {
    base: "https://cdn.jsdelivr.net/gh/jdecked/twemoji@15.1.0/assets/",
    attributes: function () {
      return { referrerpolicy: "no-referrer" };
    }
  };

  function emojify(root) {
    if (!window.twemoji || !root) return;
    twemoji.parse(root, TW);
  }

  document.querySelectorAll(".forum-cooked").forEach(emojify);

  var jump = new URLSearchParams(location.search).get("post");
  if (jump) {
    var el = document.getElementById("p-" + jump);
    if (el) {
      if (location.hash !== "#p-" + jump) {
        history.replaceState(null, "", location.pathname + location.search + "#p-" + jump);
      }
      el.scrollIntoView();
    }
  }

  function wrap(ta, before, after) {
    var s = ta.selectionStart;
    var e = ta.selectionEnd;
    var sel = ta.value.slice(s, e);
    ta.value = ta.value.slice(0, s) + before + sel + after + ta.value.slice(e);
    var inner = s + before.length;
    if (sel) {
      ta.selectionStart = ta.selectionEnd = inner + sel.length;
    } else {
      ta.selectionStart = ta.selectionEnd = inner;
    }
    ta.focus();
    ta.dispatchEvent(new Event("input"));
  }

  document.addEventListener("click", function (ev) {
    var btn = ev.target.closest(".forum-md");
    if (btn) {
      ev.preventDefault();
      var form = btn.closest("form");
      var ta = form && form.querySelector(".forum-body");
      if (ta) wrap(ta, btn.getAttribute("data-before") || "", btn.getAttribute("data-after") || "");
      return;
    }
    var reply = ev.target.closest(".forum-reply-btn");
    if (!reply) return;
    var parent = document.getElementById("forum-parent");
    var preview = document.querySelector(".forum-quote-preview");
    var ta = document.getElementById("forum-reply-body");
    if (parent) parent.value = reply.getAttribute("data-parent") || "";
    if (preview) {
      preview.hidden = false;
      preview.textContent = "Replying to " + (reply.getAttribute("data-author") || "") + " — " + (reply.getAttribute("data-excerpt") || "");
    }
    if (ta) ta.focus({ preventScroll: true });
    var box = document.getElementById("forum-reply");
    if (box) {
      var go = function () { box.scrollIntoView({ block: "end", behavior: "smooth" }); };
      go();
      setTimeout(go, 350);
    }
  });

  document.querySelectorAll(".forum-body").forEach(function (ta) {
    var form = ta.closest("form");
    var prev = form && form.querySelector(".forum-preview");
    var t;
    function run() {
      fetch("/preview", {
        method: "POST",
        credentials: "same-origin",
        headers: { "Content-Type": "application/x-www-form-urlencoded" },
        body: "content_raw=" + encodeURIComponent(ta.value)
      }).then(function (r) {
        return r.ok ? r.text() : "";
      }).then(function (html) {
        if (!prev) return;
        prev.innerHTML = html;
        emojify(prev);
      }).catch(function () {});
    }
    ta.addEventListener("input", function () {
      clearTimeout(t);
      t = setTimeout(run, 300);
    });
    run();
  });

  var box = document.createElement("ul");
  box.id = "forum-ac";
  box.className = "forum-ac";
  box.setAttribute("role", "listbox");
  box.hidden = true;
  document.body.appendChild(box);
  var acStart = 0;
  var acTimer;
  var acGen = 0;
  var acTa = null;
  var acHits = [];
  var acIdx = 0;
  var acAbort;

  function hideAc() {
    clearTimeout(acTimer);
    acGen++;
    if (acAbort) acAbort.abort();
    acTa = null;
    acHits = [];
    box.hidden = true;
    box.innerHTML = "";
  }

  function mentionToken(h) {
    var n = (h.display_name || "").trim();
    if (/^[A-Za-z0-9_]{2,32}$/.test(n)) return n;
    return h.username;
  }

  function mentionLabel(h) {
    var n = (h.display_name || "").trim() || h.username;
    if (h.username && n.toLowerCase() !== h.username.toLowerCase()) return n + " (@" + h.username + ")";
    return n;
  }

  function acButtons() {
    return box.querySelectorAll("button");
  }

  function setActive(i) {
    var buttons = acButtons();
    if (!buttons.length) return;
    acIdx = (i + buttons.length) % buttons.length;
    for (var n = 0; n < buttons.length; n++) {
      buttons[n].setAttribute("aria-selected", n === acIdx ? "true" : "false");
    }
    buttons[acIdx].scrollIntoView({ block: "nearest" });
  }

  function insertHit(ta, h) {
    var name = mentionToken(h);
    ta.value = ta.value.slice(0, acStart) + name + " " + ta.value.slice(ta.selectionStart);
    var p = acStart + name.length + 1;
    ta.selectionStart = ta.selectionEnd = p;
    ta.focus();
    hideAc();
    ta.dispatchEvent(new Event("input"));
  }

  // ponytail: visualViewport is the visible area above the keyboard. Caret coords if the list must sit on the @.
  function placeAc() {
    if (!acTa || box.hidden) return;
    var rect = acTa.getBoundingClientRect();
    var vv = window.visualViewport;
    var vTop = vv ? vv.offsetTop : 0;
    var vLeft = vv ? vv.offsetLeft : 0;
    var vH = vv ? vv.height : window.innerHeight;
    var vW = vv ? vv.width : window.innerWidth;
    var gap = 8;
    box.style.maxHeight = "none";
    var contentH = box.scrollHeight;
    var spaceBelow = vTop + vH - rect.bottom - gap;
    var spaceAbove = rect.top - vTop - gap;
    var above = spaceBelow < Math.min(contentH, 160) && spaceAbove > spaceBelow;
    var room = Math.max(48, above ? spaceAbove : spaceBelow);
    box.style.maxHeight = Math.min(room, Math.floor(vH * 0.5)) + "px";
    box.style.maxWidth = Math.max(120, Math.floor(vW - gap * 2)) + "px";
    var w = box.offsetWidth;
    var h = box.offsetHeight;
    var left = rect.left;
    if (left + w > vLeft + vW - gap) left = vLeft + vW - gap - w;
    if (left < vLeft + gap) left = vLeft + gap;
    var top = above ? rect.top - h : rect.bottom;
    if (top < vTop + gap) top = vTop + gap;
    if (top + h > vTop + vH - gap) top = Math.max(vTop + gap, vTop + vH - gap - h);
    box.style.left = left + "px";
    box.style.top = top + "px";
  }

  function onViewChange() {
    if (!box.hidden) placeAc();
  }
  window.addEventListener("resize", onViewChange);
  window.addEventListener("scroll", function (ev) {
    if (box.contains(ev.target)) return;
    onViewChange();
  }, true);
  if (window.visualViewport) {
    visualViewport.addEventListener("resize", onViewChange);
    visualViewport.addEventListener("scroll", onViewChange);
  }

  document.addEventListener("keydown", function (ev) {
    if (box.hidden || !acTa || document.activeElement !== acTa) return;
    if (ev.isComposing || ev.keyCode === 229) return;
    if (ev.key === "Escape") {
      ev.preventDefault();
      hideAc();
      return;
    }
    if (ev.key === "ArrowDown") {
      ev.preventDefault();
      setActive(acIdx + 1);
      return;
    }
    if (ev.key === "ArrowUp") {
      ev.preventDefault();
      setActive(acIdx - 1);
      return;
    }
    if (ev.key === "Enter" || ev.key === "Tab") {
      if (!acHits[acIdx]) return;
      ev.preventDefault();
      insertHit(acTa, acHits[acIdx]);
    }
  });

  document.addEventListener("input", function (ev) {
    var ta = ev.target.closest && ev.target.closest(".forum-body");
    if (!ta) return;
    var pos = ta.selectionStart;
    var before = ta.value.slice(0, pos);
    var m = before.match(/(^|[^A-Za-z0-9_])@([A-Za-z0-9_ ]{0,48})$/);
    if (!m || / $/.test(m[2])) {
      hideAc();
      return;
    }
    acStart = pos - m[2].length;
    var q = m[2];
    clearTimeout(acTimer);
    var gen = ++acGen;
    acTimer = setTimeout(function () {
      if (acAbort) acAbort.abort();
      acAbort = new AbortController();
      fetch("/forum/users?q=" + encodeURIComponent(q), { credentials: "same-origin", signal: acAbort.signal })
        .then(function (r) { return r.ok ? r.json() : Promise.reject(); })
        .then(function (hits) {
          if (gen !== acGen) return;
          box.innerHTML = "";
          acHits = Array.isArray(hits) ? hits : [];
          if (!acHits.length) {
            box.hidden = true;
            return;
          }
          acHits.forEach(function (h, i) {
            var li = document.createElement("li");
            li.setAttribute("role", "presentation");
            var b = document.createElement("button");
            b.type = "button";
            b.setAttribute("role", "option");
            b.textContent = mentionLabel(h);
            b.addEventListener("pointerdown", function (e) {
              e.preventDefault();
              insertHit(ta, h);
            });
            li.appendChild(b);
            box.appendChild(li);
            if (i === 0) b.setAttribute("aria-selected", "true");
          });
          acTa = ta;
          acIdx = 0;
          box.hidden = false;
          placeAc();
        })
        .catch(function (err) {
          if (err && err.name === "AbortError") return;
          if (gen === acGen) hideAc();
        });
    }, 150);
  });

  document.addEventListener("click", function (ev) {
    if (!box.contains(ev.target)) hideAc();
  });
})();

(function () {
  var dlg = document.getElementById("forum-full");
  if (!dlg || !dlg.showModal) return;
  var img = dlg.querySelector(".slideshow-full-img");
  var prev = dlg.querySelector(".slideshow-full-prev");
  var next = dlg.querySelector(".slideshow-full-next");
  var close = dlg.querySelector(".slideshow-full-close");
  var urls = [];
  var i = 0;
  function paint() {
    if (!urls.length) return;
    img.src = urls[i];
    prev.hidden = urls.length < 2;
    next.hidden = urls.length < 2;
  }
  function go(d) {
    if (!urls.length) return;
    i = (i + d + urls.length) % urls.length;
    paint();
  }
  document.addEventListener("click", function (ev) {
    var btn = ev.target.closest(".forum-thumbs button[data-src]");
    if (!btn) return;
    var list = btn.closest("ul");
    urls = [];
    list.querySelectorAll("button[data-src]").forEach(function (b) {
      urls.push(b.getAttribute("data-src"));
    });
    i = urls.indexOf(btn.getAttribute("data-src"));
    if (i < 0) i = 0;
    paint();
    dlg.showModal();
  });
  prev.addEventListener("click", function (e) { e.stopPropagation(); go(-1); });
  next.addEventListener("click", function (e) { e.stopPropagation(); go(1); });
  close.addEventListener("click", function () { dlg.close(); });
  dlg.addEventListener("click", function (e) { if (e.target === dlg) dlg.close(); });
  dlg.addEventListener("keydown", function (e) {
    if (e.key === "ArrowLeft") { e.preventDefault(); go(-1); }
    if (e.key === "ArrowRight") { e.preventDefault(); go(1); }
  });
  var tx = 0, ty = 0;
  dlg.addEventListener("touchstart", function (e) {
    var t = e.changedTouches[0];
    tx = t.clientX;
    ty = t.clientY;
  }, { passive: true });
  dlg.addEventListener("touchend", function (e) {
    var t = e.changedTouches[0];
    var dx = t.clientX - tx, dy = t.clientY - ty;
    if (Math.abs(dx) < 40 && Math.abs(dy) < 40) return;
    if (Math.abs(dy) > Math.abs(dx)) {
      if (dy > 40) dlg.close();
      return;
    }
    go(dx < 0 ? 1 : -1);
  });
})();

(function () {
  var MAX = 15 * 1024 * 1024;
  var OK = /^(image\/(jpeg|png|gif|webp)|audio\/(mpeg|mp3|mp4|aac|wav|x-wav|x-m4a|ogg|flac|webm)|video\/(mp4|webm|ogg)|application\/pdf)$/;
  var OKNAME = /\.(jpe?g|png|gif|webp|mp3|wav|ogg|oga|flac|m4a|aac|mp4|m4v|webm|ogv|pdf)$/;

  function hasFiles(e) {
    var t = e.dataTransfer;
    if (!t) return false;
    if (t.types) {
      if (t.types.indexOf && t.types.indexOf("Files") !== -1) return true;
      if (t.types.contains && t.types.contains("Files")) return true;
    }
    return !!(t.files && t.files.length);
  }

  function ok(f) {
    if (!f || !f.size || f.size > MAX) return false;
    var n = (f.name || "").toLowerCase();
    var t = (f.type || "").toLowerCase();
    if (/\.hei[cf]$/.test(n) || t.indexOf("heic") !== -1 || t.indexOf("heif") !== -1) return false;
    if (OK.test(t)) return true;
    if (!t || t === "application/octet-stream") return OKNAME.test(n);
    return false;
  }

  function used(form) {
    return form.querySelectorAll(".forum-existing input[name=remove_photo]:not(:checked)").length;
  }

  function names(form) {
    var input = form.querySelector('input[name="photos"]');
    var el = form.querySelector(".forum-drop-names");
    if (!input || !el) return;
    var bits = [];
    for (var i = 0; i < input.files.length; i++) bits.push(input.files[i].name);
    el.hidden = !bits.length;
    el.textContent = bits.join(", ");
  }

  function put(form, incoming, merge) {
    var input = form.querySelector('input[name="photos"]');
    if (!input || !incoming || !incoming.length) return;
    var room = 3 - used(form);
    if (room <= 0) {
      alert("Max 3 files on a post");
      return;
    }
    var keep = [];
    var i;
    if (merge) {
      for (i = 0; i < input.files.length && keep.length < room; i++) keep.push(input.files[i]);
    }
    for (i = 0; i < incoming.length && keep.length < room; i++) {
      if (!ok(incoming[i])) {
        alert("Use JPEG, PNG, GIF, WebP, audio, video, or PDF under 15MB");
        if (!merge) input.value = "";
        names(form);
        return;
      }
      keep.push(incoming[i]);
    }
    try {
      var dt = new DataTransfer();
      keep.forEach(function (f) { dt.items.add(f); });
      input.files = dt.files;
    } catch (e) {
      names(form);
      return;
    }
    names(form);
    form.classList.remove("drag");
  }

  function target(el) {
    var form = el && el.closest && el.closest("form.forum-compose");
    if (form && form.querySelector('input[name="photos"]')) return form;
    return document.querySelector("#forum-reply form.forum-compose") || document.querySelector("form.forum-compose");
  }

  document.querySelectorAll("form.forum-compose").forEach(function (form) {
    var input = form.querySelector('input[name="photos"]');
    if (!input) return;
    input.addEventListener("change", function () { put(form, input.files, false); });
  });

  document.addEventListener("dragover", function (e) {
    if (!hasFiles(e)) return;
    e.preventDefault();
    e.dataTransfer.dropEffect = "copy";
    var form = target(e.target);
    document.querySelectorAll("form.forum-compose.drag").forEach(function (f) {
      if (f !== form) f.classList.remove("drag");
    });
    if (form) form.classList.add("drag");
  });
  document.addEventListener("dragleave", function (e) {
    if (e.target === document.documentElement) {
      document.querySelectorAll("form.forum-compose.drag").forEach(function (f) { f.classList.remove("drag"); });
    }
  });
  document.addEventListener("drop", function (e) {
    if (!hasFiles(e)) return;
    e.preventDefault();
    var form = target(e.target);
    document.querySelectorAll("form.forum-compose.drag").forEach(function (f) { f.classList.remove("drag"); });
    if (!form) return;
    put(form, e.dataTransfer.files, true);
    var box = form.closest("#forum-reply") || form;
    if (box.scrollIntoView) box.scrollIntoView({ behavior: "smooth", block: "nearest" });
  });
})();
