(function () {
  var hero = document.querySelector(".hero");
  var play = document.querySelector(".hero-play");
  var video = document.querySelector(".hero video");
  var close = document.querySelector(".hero-close");
  if (!hero || !play || !video) return;
  play.addEventListener("click", function () {
    hero.classList.add("playing");
    play.hidden = true;
    if (close) close.hidden = false;
    video.play();
  });
  if (close) close.addEventListener("click", function () {
    video.pause();
    video.currentTime = 0;
    hero.classList.remove("playing");
    play.hidden = false;
    close.hidden = true;
  });
})();
