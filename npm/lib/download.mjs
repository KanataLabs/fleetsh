// SPDX-License-Identifier: GPL-3.0-only
export async function download(url, options = {}) {
  for (let attempt = 0; attempt < 2; attempt++) {
    try {
      return await downloadAttempt(url, options);
    } catch (error) {
      const transient = error.name === 'TimeoutError' || error.name === 'TypeError' || error.retryable;
      if (attempt === 1 || !transient) throw error;
    }
  }
}

async function downloadAttempt(url, { expectedSize, fetchImpl = fetch, timeoutMs = 120000, maxSize = 16 * 1024 * 1024 } = {}) {
  const signal = AbortSignal.timeout(timeoutMs);
  for (let redirects = 0; redirects <= 5; redirects++) {
    if (new URL(url).protocol !== 'https:') throw new Error('Release downloads require HTTPS');
    const response = await fetchImpl(url, { redirect: 'manual', signal });
    if ([301, 302, 303, 307, 308].includes(response.status)) {
      await response.body?.cancel();
      const location = response.headers.get('location');
      if (!location) throw new Error('Download redirect has no destination');
      url = new URL(location, url).href;
      continue;
    }
    if (!response.ok || !response.body) {
      await response.body?.cancel();
      const error = new Error('Release download failed (HTTP ' + response.status + ')');
      error.status = response.status;
      error.retryable = [408, 429].includes(response.status) || response.status >= 500;
      throw error;
    }
    const limit = Math.min(expectedSize ?? maxSize, maxSize);
    const declared = response.headers.get('content-length');
    if (declared && (!/^\d+$/.test(declared) || Number(declared) > limit)) {
      await response.body.cancel();
      throw new Error('Release download exceeds expected size');
    }
    const reader = response.body.getReader();
    const chunks = [];
    let size = 0;
    try {
      for (;;) {
        const { done, value } = await reader.read();
        if (done) break;
        size += value.byteLength;
        if (size > limit) throw new Error('Release download exceeds expected size');
        chunks.push(Buffer.from(value));
      }
    } catch (error) {
      await reader.cancel().catch(() => {});
      throw error;
    } finally {
      reader.releaseLock();
    }
    if (expectedSize !== undefined && size !== expectedSize) throw new Error('Truncated release download');
    return Buffer.concat(chunks, size);
  }
  throw new Error('Too many release download redirects');
}
