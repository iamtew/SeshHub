(function () {
  var article = document.querySelector("article.profile-feed");
  if (!article) return;
  var toggle = article.querySelector("[data-feed-toggle]");
  var feat = article.querySelector(".feed-featured");

  function ytWatch(id) {
    return "https://www.youtube.com/watch?v=" + id;
  }
  function ytEmbed(id) {
    return "https://www.youtube-nocookie.com/embed/" + id;
  }

  function setHideUI(root, hidden) {
    if (!root) return;
    root.classList.toggle("is-hidden", hidden);
    var act = root.querySelector("[data-hide-action]");
    var btn = root.querySelector("[data-hide-btn]");
    if (act) act.value = hidden ? "show" : "hide";
    if (btn) {
      var label = hidden ? "Unhide" : "Hide";
      btn.innerHTML = hidden
        ? '<i class="fa-solid fa-eye-slash" aria-hidden="true"></i> Unhide'
        : '<i class="fa-regular fa-eye" aria-hidden="true"></i> Hide';
      btn.setAttribute("aria-label", label);
      btn.setAttribute("aria-pressed", hidden ? "true" : "false");
    }
  }

  function setIds(root, id) {
    root.querySelectorAll('input[name="id"]').forEach(function (el) {
      el.value = id;
    });
  }

  function fillLi(li, id, thumb, title, stats, hidden) {
    li.setAttribute("data-id", id);
    li.setAttribute("data-thumb", thumb || "");
    li.setAttribute("data-title", title || "");
    li.setAttribute("data-stats", stats || "");
    li.classList.toggle("is-hidden", !!hidden);
    var a = li.querySelector("a");
    if (a) a.href = ytWatch(id);
    var img = li.querySelector("img");
    if (img) {
      if (thumb) img.src = thumb;
      else img.removeAttribute("src");
    }
    var strong = li.querySelector("strong");
    if (strong) strong.textContent = title || "";
    var who = li.querySelector(".who");
    if (who) {
      who.textContent = stats || "";
      who.hidden = !stats;
    }
    setIds(li, id);
    setHideUI(li, !!hidden);
    var star = li.querySelector('button[aria-label="Set as featured"]');
    if (star) {
      star.removeAttribute("aria-pressed");
      star.innerHTML = '<i class="fa-regular fa-star" aria-hidden="true"></i>';
    }
  }

  function toGrid(box) {
    var id = box.getAttribute("data-id");
    if (!id) return;
    var ul = article.querySelector("ul.grid");
    var h2 = article.querySelector("h2");
    if (!ul) {
      ul = document.createElement("ul");
      ul.className = "grid";
      if (!h2 || h2.textContent !== "Videos") {
        h2 = document.createElement("h2");
        h2.textContent = "Videos";
        box.after(h2);
      }
      h2.after(ul);
    }
    var li = (ul.querySelector("li") || document.createElement("li")).cloneNode(true);
    if (!li.querySelector("a")) return;
    fillLi(li, id, box.getAttribute("data-thumb"), box.getAttribute("data-title"), box.getAttribute("data-stats"), box.classList.contains("is-hidden"));
    ul.prepend(li);
  }

  function setFeaturedFrom(el) {
    if (!feat) return;
    var id = el.getAttribute("data-id");
    var thumb = el.getAttribute("data-thumb") || "";
    var title = el.getAttribute("data-title") || "";
    var stats = el.getAttribute("data-stats") || "";
    var hidden = el.classList.contains("is-hidden");
    if (feat.getAttribute("data-id") && feat.getAttribute("data-id") !== id) toGrid(feat);
    feat.hidden = false;
    feat.setAttribute("data-id", id);
    feat.setAttribute("data-thumb", thumb);
    feat.setAttribute("data-title", title);
    feat.setAttribute("data-stats", stats);
    var iframe = feat.querySelector("iframe");
    iframe.src = ytEmbed(id);
    iframe.title = title;
    setIds(feat, id);
    setHideUI(feat, hidden);
    var star = feat.querySelector('button[aria-label="Set as featured"]');
    if (star) {
      star.setAttribute("aria-pressed", "true");
      star.innerHTML = '<i class="fa-solid fa-star" aria-hidden="true"></i>';
    }
    if (el.tagName === "LI") el.remove();
  }

  if (toggle) {
    toggle.addEventListener("click", function () {
      var on = !article.classList.contains("feed-edit");
      article.classList.toggle("feed-edit", on);
      toggle.setAttribute("aria-pressed", on ? "true" : "false");
      toggle.innerHTML = on
        ? '<i class="fa-solid fa-check" aria-hidden="true"></i> Done'
        : '<i class="fa-solid fa-pen-to-square" aria-hidden="true"></i> Edit YouTube feed';
    });
  }

  article.addEventListener("submit", function (e) {
    var form = e.target;
    if (!form.closest(".clip-tools")) return;
    e.preventDefault();
    var body = new URLSearchParams(new FormData(form));
    fetch(form.getAttribute("action"), {
      method: "POST",
      headers: { Accept: "application/json", "Content-Type": "application/x-www-form-urlencoded" },
      body: body,
      credentials: "same-origin"
    }).then(function (r) {
      if (!r.ok) return;
      var id = body.get("id");
      var op = body.get("op");
      var root = form.closest("[data-id]");
      if (op === "feature") {
        setFeaturedFrom(root);
        return;
      }
      if (op === "hide" || op === "show") {
        setHideUI(root, op === "hide");
        if (feat && feat.getAttribute("data-id") === id) setHideUI(feat, op === "hide");
        var li = article.querySelector('li[data-id="' + id + '"]');
        if (li && li !== root) setHideUI(li, op === "hide");
      }
    });
  });
})();
