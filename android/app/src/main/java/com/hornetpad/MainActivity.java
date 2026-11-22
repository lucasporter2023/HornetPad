package com.hornetpad;

import android.annotation.SuppressLint;
import android.app.Activity;
import android.graphics.Color;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.view.HapticFeedbackConstants;
import android.view.View;
import android.view.Window;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;
import android.webkit.JavascriptInterface;
import android.webkit.WebResourceError;
import android.webkit.WebResourceRequest;
import android.webkit.WebSettings;
import android.webkit.WebView;
import android.webkit.WebViewClient;

/**
 * HornetPad tablet client: a full-screen, always-on WebView that shows the
 * cockpit page served by HornetPad.exe. Over USB the PC is reachable at
 * 127.0.0.1 because HornetPad.exe runs "adb reverse tcp:8765 tcp:8765".
 */
public class MainActivity extends Activity {

    private static final String SERVER = "http://127.0.0.1:8765/";
    private static final long RETRY_MS = 2000;

    private WebView web;
    private final Handler handler = new Handler(Looper.getMainLooper());
    private boolean showingOffline = false;

    private final Runnable retry = new Runnable() {
        @Override public void run() {
            if (web != null) web.loadUrl(SERVER);
        }
    };

    @SuppressLint("SetJavaScriptEnabled")
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        Window win = getWindow();
        win.addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        WindowManager.LayoutParams lp = win.getAttributes();
        lp.layoutInDisplayCutoutMode = WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
        win.setAttributes(lp);

        web = new WebView(this);
        web.setBackgroundColor(Color.BLACK);
        web.setHapticFeedbackEnabled(true);
        web.setOverScrollMode(View.OVER_SCROLL_NEVER);
        web.setVerticalScrollBarEnabled(false);
        web.setHorizontalScrollBarEnabled(false);

        WebSettings s = web.getSettings();
        s.setJavaScriptEnabled(true);
        s.setDomStorageEnabled(true);
        s.setCacheMode(WebSettings.LOAD_NO_CACHE);
        s.setMediaPlaybackRequiresUserGesture(false);
        s.setSupportZoom(false);
        s.setBuiltInZoomControls(false);
        s.setTextZoom(100);
        s.setUserAgentString(s.getUserAgentString() + " HornetPadApp/1.0");

        web.addJavascriptInterface(new Bridge(), "HornetPadNative");
        web.setWebViewClient(new WebViewClient() {
            @Override
            public void onReceivedError(WebView view, WebResourceRequest request, WebResourceError error) {
                if (request.isForMainFrame()) showOffline();
            }

            @Override
            public void onPageFinished(WebView view, String url) {
                if (url != null && url.startsWith(SERVER)) showingOffline = false;
            }
        });

        setContentView(web);
        hideSystemBars();
        web.loadUrl(SERVER);
    }

    /** Shown while HornetPad.exe is not reachable; retries automatically. */
    private void showOffline() {
        if (!showingOffline) {
            showingOffline = true;
            String html = "<html><body style='background:#000;color:#8a9;font-family:sans-serif;"
                    + "display:flex;align-items:center;justify-content:center;height:100vh;margin:0;"
                    + "text-align:center'><div><div style='font-size:28px;color:#5fd35f'>HornetPad</div>"
                    + "<p>Waiting for HornetPad.exe on the PC&hellip;</p>"
                    + "<p style='color:#567;font-size:14px'>Plug in the USB cable, allow USB debugging,"
                    + " and start HornetPad.exe.<br>This screen reconnects by itself.</p></div></body></html>";
            web.loadDataWithBaseURL(null, html, "text/html", "utf-8", null);
        }
        handler.removeCallbacks(retry);
        handler.postDelayed(retry, RETRY_MS);
    }

    private void hideSystemBars() {
        if (android.os.Build.VERSION.SDK_INT >= 30) {
            WindowInsetsController c = getWindow().getInsetsController();
            if (c != null) {
                c.hide(WindowInsets.Type.statusBars() | WindowInsets.Type.navigationBars());
                c.setSystemBarsBehavior(WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
            }
        } else {
            getWindow().getDecorView().setSystemUiVisibility(
                    View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
                            | View.SYSTEM_UI_FLAG_FULLSCREEN
                            | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
                            | View.SYSTEM_UI_FLAG_LAYOUT_STABLE
                            | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
                            | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN);
        }
    }

    @Override
    public void onWindowFocusChanged(boolean hasFocus) {
        super.onWindowFocusChanged(hasFocus);
        if (hasFocus) hideSystemBars();
    }

    @Override
    protected void onResume() {
        super.onResume();
        if (web != null) web.onResume();
        hideSystemBars();
    }

    @Override
    protected void onPause() {
        if (web != null) web.onPause();
        super.onPause();
    }

    @Override
    protected void onDestroy() {
        handler.removeCallbacks(retry);
        if (web != null) {
            web.destroy();
            web = null;
        }
        super.onDestroy();
    }

    @SuppressWarnings("deprecation")
    @Override
    public void onBackPressed() {
        // Ignore Back so a stray swipe mid-flight doesn't close the panel.
    }

    /** Called from the page: window.HornetPadNative.haptic() */
    private class Bridge {
        @JavascriptInterface
        public void haptic() {
            handler.post(new Runnable() {
                @Override public void run() {
                    if (web != null) web.performHapticFeedback(HapticFeedbackConstants.VIRTUAL_KEY);
                }
            });
        }
    }
}
