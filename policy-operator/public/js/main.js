// Dark mode toggle
(function () {
  var html = document.documentElement;
  var btn  = document.getElementById('theme-toggle');
  if (!btn) return;
  btn.addEventListener('click', function () {
    var next = html.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
    html.setAttribute('data-theme', next);
    localStorage.setItem('theme', next);
  });
})();

// Mobile nav toggle
(function () {
  const toggle = document.getElementById('navbar-toggle');
  const menu   = document.getElementById('navbar-menu');
  if (toggle && menu) {
    toggle.addEventListener('click', function () {
      const open = menu.classList.toggle('is-open');
      toggle.setAttribute('aria-expanded', open);
    });
    // Close on outside click
    document.addEventListener('click', function (e) {
      if (!toggle.contains(e.target) && !menu.contains(e.target)) {
        menu.classList.remove('is-open');
        toggle.setAttribute('aria-expanded', 'false');
      }
    });
  }
})();

// Mobile sidebar toggle
(function () {
  const btn     = document.getElementById('sidebar-toggle');
  const sidebar = document.getElementById('doc-sidebar');
  if (btn && sidebar) {
    btn.addEventListener('click', function () {
      sidebar.classList.toggle('is-open');
    });
  }
})();

// Code copy buttons
function initCopyButtons(root) {
  (root || document).querySelectorAll('pre:not(.code-wrapper > pre)').forEach(function (pre) {
    var wrapper = document.createElement('div');
    wrapper.className = 'code-wrapper';
    pre.parentNode.insertBefore(wrapper, pre);
    wrapper.appendChild(pre);

    var btn = document.createElement('button');
    btn.className = 'code-copy';
    btn.textContent = 'Copy';
    btn.setAttribute('aria-label', 'Copy code');
    wrapper.appendChild(btn);
  });
}

// Delegated click handler for all copy buttons (works for dynamically added content)
document.addEventListener('click', function (e) {
  var btn = e.target.closest('.code-copy');
  if (!btn) return;
  var pre = btn.closest('.code-wrapper') && btn.closest('.code-wrapper').querySelector('pre');
  if (!pre) return;
  var text = (pre.querySelector('code') || pre).textContent;
  navigator.clipboard.writeText(text).then(function () {
    btn.textContent = 'Copied!';
    setTimeout(function () { btn.textContent = 'Copy'; }, 2000);
  });
});

initCopyButtons();

// ── API Schema: tabs + field-tree accordions ─────────────────────────────

// Wire api-schema tabs and move adjacent api-examples content into examples panel
(function () {
  document.querySelectorAll('.api-schema').forEach(function (schema) {

    // ── Tabs ──
    schema.querySelectorAll('.api-schema__tab').forEach(function (tab) {
      tab.addEventListener('click', function () {
        schema.querySelectorAll('.api-schema__tab').forEach(function (t) {
          t.classList.remove('api-schema__tab--active');
          t.setAttribute('aria-selected', 'false');
        });
        schema.querySelectorAll('.api-schema__panel').forEach(function (p) {
          p.hidden = true;
        });
        tab.classList.add('api-schema__tab--active');
        tab.setAttribute('aria-selected', 'true');
        var target = schema.querySelector('[data-schema-panel="' + tab.dataset.schemaTab + '"]');
        if (target) target.hidden = false;
      });
    });

    // ── Move adjacent api-examples-src content into examples panel ──
    var exPanel = schema.querySelector('[data-schema-panel="examples"]');
    if (!exPanel) return;

    // Search the next few siblings for the source node
    var sibling = schema.nextElementSibling;
    while (sibling) {
      var src = sibling.querySelector('.api-examples-src');
      if (!src) src = sibling.classList.contains('api-examples-src') ? sibling : null;
      if (src) {
        exPanel.innerHTML = src.innerHTML;
        initCopyButtons(exPanel);
        sibling.remove();
        break;
      }
      // Stop searching after a non-schema element that isn't just whitespace
      if (!sibling.classList.contains('api-examples-src')) break;
      sibling = sibling.nextElementSibling;
    }
  });
})();

// Field-tree accordion toggle (delegated — works for dynamically nested rows)
document.addEventListener('click', function (e) {
  var btn = e.target.closest('[data-api-toggle]');
  if (!btn) return;
  var expanded = btn.getAttribute('aria-expanded') === 'true';
  var bodyId   = btn.getAttribute('aria-controls');
  var body     = document.getElementById(bodyId);
  if (!body) return;
  btn.setAttribute('aria-expanded', String(!expanded));
  body.hidden = expanded;
});

// Active TOC link on scroll
(function () {
  var tocLinks  = Array.from(document.querySelectorAll('.toc-link'));
  var headings  = Array.from(document.querySelectorAll('.doc-body h2[id], .doc-body h3[id]'));
  if (!tocLinks.length || !headings.length) return;

  var observer = new IntersectionObserver(function (entries) {
    entries.forEach(function (entry) {
      if (entry.isIntersecting) {
        tocLinks.forEach(function (l) { l.classList.remove('active'); });
        var active = document.querySelector('.toc-link[href="#' + entry.target.id + '"]');
        if (active) active.classList.add('active');
      }
    });
  }, { rootMargin: '-' + (64 + 24) + 'px 0px -70% 0px' });

  headings.forEach(function (h) { observer.observe(h); });
})();
