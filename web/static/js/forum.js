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
    if (ta) ta.focus();
    var box = document.getElementById("forum-reply");
    if (box) box.scrollIntoView({ behavior: "smooth" });
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
  box.className = "forum-ac";
  box.hidden = true;
  document.body.appendChild(box);
  var acStart = 0;
  var acTimer;
  var acGen = 0;
  var acTa = null;

  function hideAc() {
    clearTimeout(acTimer);
    acGen++;
    acTa = null;
    box.hidden = true;
    box.innerHTML = "";
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

  document.addEventListener("input", function (ev) {
    var ta = ev.target.closest && ev.target.closest(".forum-body");
    if (!ta) return;
    var pos = ta.selectionStart;
    var before = ta.value.slice(0, pos);
    var m = before.match(/(^|[^A-Za-z0-9_])@([A-Za-z0-9_]{0,32})$/);
    if (!m) {
      hideAc();
      return;
    }
    acStart = pos - m[2].length;
    var q = m[2];
    clearTimeout(acTimer);
    var gen = ++acGen;
    acTimer = setTimeout(function () {
      fetch("/forum/users?q=" + encodeURIComponent(q), { credentials: "same-origin" })
        .then(function (r) { return r.json(); })
        .then(function (hits) {
          if (gen !== acGen) return;
          box.innerHTML = "";
          if (!hits || !hits.length) {
            box.hidden = true;
            return;
          }
          hits.forEach(function (h) {
            var li = document.createElement("li");
            var b = document.createElement("button");
            b.type = "button";
            b.textContent = h.username + (h.display_name && h.display_name !== h.username ? " (" + h.display_name + ")" : "");
            function pick(ev) {
              ev.preventDefault();
              if (b.disabled) return;
              b.disabled = true;
              ta.value = ta.value.slice(0, acStart) + h.username + " " + ta.value.slice(ta.selectionStart);
              var p = acStart + h.username.length + 1;
              ta.selectionStart = ta.selectionEnd = p;
              ta.focus();
              hideAc();
              ta.dispatchEvent(new Event("input"));
            }
            b.addEventListener("pointerdown", pick);
            b.addEventListener("click", pick);
            li.appendChild(b);
            box.appendChild(li);
          });
          acTa = ta;
          box.hidden = false;
          placeAc();
        })
        .catch(function () {
          if (gen === acGen) hideAc();
        });
    }, 150);
  });

  document.addEventListener("click", function (ev) {
    if (!box.contains(ev.target)) hideAc();
  });
})();
