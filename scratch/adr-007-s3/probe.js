window.__v = []; window.__errs = [];
document.addEventListener('securitypolicyviolation', e => window.__v.push(e.violatedDirective + ' ' + (e.blockedURI || '') + ' ' + (e.sample || '')));
window.addEventListener('error', e => window.__errs.push(e.message));
