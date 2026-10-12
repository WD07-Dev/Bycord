package com.ourgram.bycord;

import android.app.Activity;
import android.app.LocaleManager;
import android.content.Context;
import android.content.Intent;
import android.content.res.ColorStateList;
import android.content.res.Configuration;
import android.net.VpnService;
import android.os.Build;
import android.os.Bundle;
import android.os.LocaleList;
import android.view.View;
import android.widget.Button;
import android.widget.EditText;
import android.widget.LinearLayout;
import android.widget.TextView;
import android.widget.Toast;
import com.ourgram.bycord.core.mobile.Mobile;
import java.util.Locale;
import org.json.JSONObject;

public final class MainActivity extends Activity {
    private static final int VPN_REQUEST = 10;
    private static final int IDLE = 0xff8e9991;
    private static final int WORKING = 0xffe7ba48;
    private static final int SUCCESS = 0xffc6ff3d;
    private static final int FAILURE = 0xffff626b;
    private static final int ACTIVE = 0xffc6ff3d;
    private static final int INACTIVE = 0xff171b18;
    private static final int ACTIVE_TEXT = 0xff0a0d0a;
    private static final int INACTIVE_TEXT = 0xffeef3ef;
    private static final String SETTINGS = "settings";

    private EditText host;
    private EditText port;
    private EditText exclude;
    private TextView status;
    private LinearLayout manualFields;
    private Button autoButton;
    private Button manualButton;
    private Button startButton;
    private Button koreanButton;
    private Button englishButton;
    private boolean autoMode = true;

    @Override
    protected void attachBaseContext(Context base) {
        String language = base.getSharedPreferences(SETTINGS, MODE_PRIVATE).getString("language", "");
        if(language.isEmpty()) {
            super.attachBaseContext(base);
            return;
        }
        Configuration configuration = new Configuration(base.getResources().getConfiguration());
        configuration.setLocale(Locale.forLanguageTag(language));
        super.attachBaseContext(base.createConfigurationContext(configuration));
    }

    @Override
    public void onCreate(Bundle state) {
        super.onCreate(state);
        setContentView(R.layout.activity_main);

        TextView versionLabel = findViewById(R.id.versionLabel);
        versionLabel.setText(getString(R.string.version_format, BuildConfig.VERSION_NAME));

        host = findViewById(R.id.host);
        port = findViewById(R.id.port);
        exclude = findViewById(R.id.exclude);
        status = findViewById(R.id.status);
        manualFields = findViewById(R.id.manualFields);
        autoButton = findViewById(R.id.autoButton);
        manualButton = findViewById(R.id.manualButton);
        startButton = findViewById(R.id.startButton);
        koreanButton = findViewById(R.id.koreanButton);
        englishButton = findViewById(R.id.englishButton);
        Button stopButton = findViewById(R.id.stopButton);

        if(getPreferences(MODE_PRIVATE).contains("exclude")) {
            exclude.setText(getPreferences(MODE_PRIVATE).getString("exclude", "KR"));
        }else {
            exclude.setText("");
            startButton.setEnabled(false);
            new Thread(this::loadDefaultCountry, "BycordCountryLookup").start();
        }
        host.setText(getPreferences(MODE_PRIVATE).getString("host", ""));
        port.setText(getPreferences(MODE_PRIVATE).getString("port", "1080"));

        koreanButton.setOnClickListener(view -> setLanguage("ko"));
        englishButton.setOnClickListener(view -> setLanguage("en"));
        autoButton.setOnClickListener(view -> setAutoMode(true));
        manualButton.setOnClickListener(view -> setAutoMode(false));
        startButton.setOnClickListener(view -> connect());
        stopButton.setOnClickListener(view -> stopVpn());
        updateLanguageButtons();
        setAutoMode(true);
    }

    private void loadDefaultCountry() {
        String country = "KR";
        try {
            country = Mobile.countryISO();
        } catch (Exception ignored) {
        }
        String selected = country;
        runOnUiThread(() -> {
            if(isDestroyed()) return;
            if(exclude.getText().toString().trim().isEmpty()) exclude.setText(selected);
            startButton.setEnabled(true);
        });
    }

    private void setLanguage(String language) {
        getSharedPreferences(SETTINGS, MODE_PRIVATE).edit().putString("language", language).apply();
        if(Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            LocaleManager manager = getSystemService(LocaleManager.class);
            manager.setApplicationLocales(LocaleList.forLanguageTags(language));
            return;
        }
        recreate();
    }

    private void updateLanguageButtons() {
        String language = getResources().getConfiguration().getLocales().get(0).getLanguage();
        boolean korean = language.equals("ko");
        koreanButton.setBackgroundTintList(ColorStateList.valueOf(korean ? ACTIVE : INACTIVE));
        englishButton.setBackgroundTintList(ColorStateList.valueOf(korean ? INACTIVE : ACTIVE));
        koreanButton.setTextColor(korean ? ACTIVE_TEXT : INACTIVE_TEXT);
        englishButton.setTextColor(korean ? INACTIVE_TEXT : ACTIVE_TEXT);
    }

    private void setAutoMode(boolean automatic) {
        autoMode = automatic;
        manualFields.setVisibility(automatic ? View.GONE : View.VISIBLE);
        startButton.setText(automatic ? R.string.connect_auto : R.string.connect_manual);
        autoButton.setBackgroundTintList(ColorStateList.valueOf(automatic ? ACTIVE : INACTIVE));
        manualButton.setBackgroundTintList(ColorStateList.valueOf(automatic ? INACTIVE : ACTIVE));
        autoButton.setTextColor(automatic ? ACTIVE_TEXT : INACTIVE_TEXT);
        manualButton.setTextColor(automatic ? INACTIVE_TEXT : ACTIVE_TEXT);
    }

    private void connect() {
        String excluded = exclude.getText().toString().trim();
        if(!Mobile.validExcluded(excluded)) {
            Toast.makeText(this, R.string.invalid_excluded_country, Toast.LENGTH_SHORT).show();
            return;
        }

        getPreferences(MODE_PRIVATE).edit().putString("exclude", excluded).apply();
        if(!autoMode) {
            if(validManualProxy()) requestVpnPermission();
            return;
        }

        setSearchEnabled(false);
        showStatus(getString(R.string.status_searching), WORKING);
        new Thread(() -> findProxy(excluded), "BycordProxySearch").start();
    }

    private void findProxy(String excluded) {
        try {
            JSONObject result = new JSONObject(Mobile.findProxy(excluded));
            runOnUiThread(() -> useProxy(result));
        } catch (Exception error) {
            runOnUiThread(() -> {
                if(isDestroyed()) return;
                String message = error.getMessage();
                if(message == null) message = error.getClass().getSimpleName();
                showStatus(getString(R.string.status_search_failed, message), FAILURE);
                setSearchEnabled(true);
            });
        }
    }

    private void useProxy(JSONObject result) {
        if(isDestroyed()) return;
        host.setText(result.optString("host"));
        port.setText(Integer.toString(result.optInt("port")));
        showStatus(getString(
            R.string.status_proxy_selected,
            result.optString("country"),
            result.optLong("latency_ms")
        ), SUCCESS);
        setSearchEnabled(true);
        requestVpnPermission();
    }

    private void setSearchEnabled(boolean enabled) {
        startButton.setEnabled(enabled);
        autoButton.setEnabled(enabled);
        manualButton.setEnabled(enabled);
    }

    private boolean validManualProxy() {
        String hostValue = host.getText().toString().trim();
        int portValue;
        try {
            portValue = Integer.parseInt(port.getText().toString());
        } catch (NumberFormatException error) {
            portValue = 0;
        }

        if(hostValue.isEmpty() || portValue < 1 || portValue > 65535) {
            Toast.makeText(this, R.string.invalid_proxy, Toast.LENGTH_SHORT).show();
            return false;
        }
        return true;
    }

    private void requestVpnPermission() {
        if(!validManualProxy()) return;
        Intent permission = VpnService.prepare(this);
        if(permission == null) {
            startVpn();
            return;
        }
        startActivityForResult(permission, VPN_REQUEST);
    }

    @Override
    protected void onActivityResult(int requestCode, int resultCode, Intent data) {
        super.onActivityResult(requestCode, resultCode, data);
        if(requestCode == VPN_REQUEST && resultCode == RESULT_OK) startVpn();
    }

    private void startVpn() {
        String hostValue = host.getText().toString().trim();
        int portValue = Integer.parseInt(port.getText().toString());
        getPreferences(MODE_PRIVATE).edit()
        .putString("host", hostValue)
        .putString("port", Integer.toString(portValue))
        .apply();

        Intent intent = new Intent(this, BycordVpnService.class)
        .setAction(BycordVpnService.ACTION_START)
        .putExtra(BycordVpnService.EXTRA_HOST, hostValue)
        .putExtra(BycordVpnService.EXTRA_PORT, portValue);

        startForegroundService(intent);
        showStatus(getString(R.string.status_vpn_starting), WORKING);
        status.postDelayed(() -> updateVpnStatus(10), 500);
    }

    private void stopVpn() {
        Intent intent = new Intent(this, BycordVpnService.class).setAction(BycordVpnService.ACTION_STOP);
        startService(intent);
        showStatus(getString(R.string.status_disconnected), IDLE);
    }

    private void updateVpnStatus(int attemptsLeft) {
        if(BycordVpnService.isConnected()) {
            showStatus(getString(R.string.status_vpn_connected), SUCCESS);
            return;
        }
        if(BycordVpnService.isStarting() && attemptsLeft > 0) {
            status.postDelayed(() -> updateVpnStatus(attemptsLeft - 1), 500);
            return;
        }

        String error = BycordVpnService.getLastError();
        String message = error == null
            ? getString(R.string.status_vpn_failed)
            : getString(R.string.status_vpn_failed_detail, error);
        showStatus(message, FAILURE);
    }

    private void showStatus(String message, int color) {
        status.setText(message);
        status.setTextColor(color);
    }
}