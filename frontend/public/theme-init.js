(function() {
  var stored = localStorage.getItem('theme');
  var prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  var theme = stored || (prefersDark ? 'dark' : 'light');
  if (theme === 'dark') {
    document.documentElement.classList.add('dark');
    document.documentElement.style.background = 'oklch(0.141 0.005 285.823)';
  }
})();
