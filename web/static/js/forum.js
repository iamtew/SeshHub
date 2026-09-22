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

  function hideAc() {
    box.hidden = true;
    box.innerHTML = "";
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
    acTimer = setTimeout(function () {
      fetch("/forum/users?q=" + encodeURIComponent(q), { credentials: "same-origin" })
        .then(function (r) { return r.json(); })
        .then(function (hits) {
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
            b.addEventListener("click", function () {
              ta.value = ta.value.slice(0, acStart) + h.username + " " + ta.value.slice(ta.selectionStart);
              var p = acStart + h.username.length + 1;
              ta.selectionStart = ta.selectionEnd = p;
              ta.focus();
              hideAc();
              ta.dispatchEvent(new Event("input"));
            });
            li.appendChild(b);
            box.appendChild(li);
          });
          var r = ta.getBoundingClientRect();
          box.style.left = window.scrollX + r.left + "px";
          box.style.top = window.scrollY + r.bottom + "px";
          box.hidden = false;
        })
        .catch(hideAc);
    }, 150);
  });

  document.addEventListener("click", function (ev) {
    if (!box.contains(ev.target)) hideAc();
  });
})();
