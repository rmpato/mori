/* mori — the behaviour this page needs.
 *
 * The same three pieces tuki's page has — the topbar, the copy button and the
 * install tabs — lifted from it so the two pages behave identically. tuki's
 * file also drives its interactive demo; mori has screenshots instead. */

(function () {
  "use strict";

  (function topbar() {
      var bar = document.getElementById("topbar");
      var hero = document.querySelector(".hero");
      if (!bar || !hero) return;

      var links = {};
      Array.prototype.forEach.call(bar.querySelectorAll("[href^='#']"), function (a) {
        links[a.getAttribute("href").slice(1)] = a;
      });

      // Show the bar once the hero is out of the way.
      if ("IntersectionObserver" in window) {
        new IntersectionObserver(function (entries) {
          bar.classList.toggle("show", !entries[0].isIntersecting);
        }, { rootMargin: "-60px 0px 0px 0px" }).observe(hero);
      } else {
        bar.classList.add("show");
      }

      var sections = ["look", "install", "using", "why", "tuki"]
        .map(function (id) { return document.getElementById(id); })
        .filter(Boolean);

      if (!("IntersectionObserver" in window) || !sections.length) return;

      // Whichever section crosses the middle band of the viewport is "current".
      var visible = {};
      var io = new IntersectionObserver(function (entries) {
        entries.forEach(function (e) { visible[e.target.id] = e.isIntersecting; });

        var current = null;
        sections.forEach(function (s) {
          if (visible[s.id] && !current) current = s.id;
        });

        Object.keys(links).forEach(function (id) {
          links[id].removeAttribute("aria-current");
        });
        if (current && links[current]) {
          links[current].setAttribute("aria-current", "true");
        }
      }, { rootMargin: "-45% 0px -45% 0px" });

      sections.forEach(function (s) { io.observe(s); });
    })();

    // Copy button on the install command.
    Array.prototype.forEach.call(document.querySelectorAll(".copy"), function (btn) {
      btn.addEventListener("click", function () {
        var target = document.querySelector(btn.dataset.copy);
        if (!target) return;
        var text = target.textContent;

        var done = function () {
          var was = btn.textContent;
          btn.textContent = "copied";
          btn.classList.add("done");
          setTimeout(function () {
            btn.textContent = was;
            btn.classList.remove("done");
          }, 1600);
        };

        if (navigator.clipboard && navigator.clipboard.writeText) {
          navigator.clipboard.writeText(text).then(done, fallback);
        } else {
          fallback();
        }

        function fallback() {
          var ta = document.createElement("textarea");
          ta.value = text;
          ta.style.position = "fixed";
          ta.style.opacity = "0";
          document.body.appendChild(ta);
          ta.select();
          try { document.execCommand("copy"); done(); } catch (err) { /* nothing to do */ }
          document.body.removeChild(ta);
        }
      });
    });

    // Install tabs.
    Array.prototype.forEach.call(document.querySelectorAll("[data-tabs]"), function (group) {
      var tabs   = group.querySelectorAll("[data-tab]");
      var panels = group.querySelectorAll("[data-panel]");

      Array.prototype.forEach.call(tabs, function (tab) {
        tab.addEventListener("click", function () {
          Array.prototype.forEach.call(tabs, function (t) {
            t.setAttribute("aria-selected", String(t === tab));
          });
          Array.prototype.forEach.call(panels, function (p) {
            p.hidden = p.dataset.panel !== tab.dataset.tab;
          });
        });
      });
    });
  })();
