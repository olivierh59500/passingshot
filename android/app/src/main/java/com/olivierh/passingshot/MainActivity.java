package com.olivierh.passingshot;

import android.app.Activity;
import android.os.Build;
import android.os.Bundle;
import android.view.View;
import android.view.WindowInsets;
import android.view.WindowInsetsController;
import android.view.WindowManager;

import com.olivierh.passingshot.mobile.EbitenView;
import com.olivierh.passingshot.mobile.Mobile;

import go.Seq;

// Android owns the view lifecycle; gameplay and controls stay in the Go app.
@SuppressWarnings("deprecation")
public final class MainActivity extends Activity {
    private EbitenView gameView;
    private boolean gameSuspended;

    @Override
    protected void onCreate(Bundle state) {
        super.onCreate(state);
        Seq.setContext(getApplicationContext());
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.P) {
            WindowManager.LayoutParams attributes = getWindow().getAttributes();
            attributes.layoutInDisplayCutoutMode =
                    WindowManager.LayoutParams.LAYOUT_IN_DISPLAY_CUTOUT_MODE_SHORT_EDGES;
            getWindow().setAttributes(attributes);
        }

        gameView = new EbitenView(this);
        gameView.setFocusable(true);
        gameView.setFocusableInTouchMode(true);
        setContentView(gameView);
        gameView.requestFocus();
        hideSystemBars();
        if (Build.VERSION.SDK_INT >= 33) {
            getOnBackInvokedDispatcher().registerOnBackInvokedCallback(
                    android.window.OnBackInvokedDispatcher.PRIORITY_DEFAULT,
                    () -> Mobile.back());
        }
    }

    @Override
    protected void onPause() {
        Mobile.suspend();
        if (gameView != null && !gameSuspended) {
            gameView.suspendGame();
            gameSuspended = true;
        }
        super.onPause();
    }

    @Override
    protected void onResume() {
        super.onResume();
        hideSystemBars();
        if (gameView != null && gameSuspended) {
            gameView.resumeGame();
            gameSuspended = false;
        }
    }

    @Override
    public void onWindowFocusChanged(boolean focused) {
        super.onWindowFocusChanged(focused);
        if (focused && gameView != null) {
            hideSystemBars();
        }
    }

    @Override
    public void onBackPressed() {
        Mobile.back();
    }

    private void hideSystemBars() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
            getWindow().setDecorFitsSystemWindows(false);
            WindowInsetsController controller = getWindow().getInsetsController();
            if (controller != null) {
                controller.hide(WindowInsets.Type.statusBars() | WindowInsets.Type.navigationBars());
                controller.setSystemBarsBehavior(
                        WindowInsetsController.BEHAVIOR_SHOW_TRANSIENT_BARS_BY_SWIPE);
            }
        } else {
            getWindow().getDecorView().setSystemUiVisibility(
                    View.SYSTEM_UI_FLAG_FULLSCREEN | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION
                            | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN
                            | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
        }
    }
}
