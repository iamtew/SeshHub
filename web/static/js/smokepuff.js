(function () {
  if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) return;
  var sound = new Audio("/static/snd/smokepuff.ogg");
  document.addEventListener("click", function (e) {
    if (e.target.closest("a, button, input, textarea, select, span")) return;
    var puff = document.createElement("div");
    puff.className = "smoke-puff";
    puff.style.setProperty("--drift", (Math.random() * 40 - 20) + "px");
    puff.style.left = e.clientX + "px";
    puff.style.top = e.clientY + "px";
    document.body.appendChild(puff);
    sound.currentTime = 0;
    sound.play().catch(function () {});
    setTimeout(function () { puff.remove(); }, 700);
  });
})();
