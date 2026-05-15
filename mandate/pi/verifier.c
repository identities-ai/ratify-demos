/*
 * Mandate Demo — Physical Access Verifier
 * Raspberry Pi + Ratify C SDK
 *
 * Reads a Ratify ProofBundle JSON from a file (or stdin), verifies it
 * offline using the Ratify C SDK, and controls a GPIO LED based on the result.
 *
 * No network access required. Verification completes in <1ms.
 *
 * Build:
 *   make          (on the Pi, after building the C SDK with `cargo build --release`)
 *
 * Usage:
 *   BUNDLE_FILE=bundle.json ./verifier
 *   cat bundle.json | ./verifier            # reads stdin if BUNDLE_FILE not set
 *   BUNDLE_FILE=bundle.json NO_GPIO=1 ./verifier   # skip GPIO (laptop testing)
 *
 * GPIO wiring (BCM numbering):
 *   Pin 18 → 220Ω resistor → green LED → GND
 *   Pin 23 → 220Ω resistor → red LED   → GND
 */

#include "ratify.h"

#define _POSIX_C_SOURCE 200809L
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#define SCOPE_PHYSICAL_ACCESS "custom:physical:access"

#define GPIO_LED_GREEN 18
#define GPIO_LED_RED   23

/* ---- GPIO via sysfs ---- */

static int gpio_no = 0;  /* set to 1 when GPIO is initialised */

static void gpio_export(int pin) {
    char path[64];
    FILE *f;

    snprintf(path, sizeof(path), "/sys/class/gpio/gpio%d", pin);
    if (access(path, F_OK) == 0) return;  /* already exported */

    f = fopen("/sys/class/gpio/export", "w");
    if (!f) return;
    fprintf(f, "%d", pin);
    fclose(f);
    sleep(1);  /* settle */

    snprintf(path, sizeof(path), "/sys/class/gpio/gpio%d/direction", pin);
    f = fopen(path, "w");
    if (!f) return;
    fprintf(f, "out");
    fclose(f);
}

static void gpio_write(int pin, int value) {
    char path[64];
    snprintf(path, sizeof(path), "/sys/class/gpio/gpio%d/value", pin);
    FILE *f = fopen(path, "w");
    if (!f) return;
    fprintf(f, "%d", value);
    fclose(f);
}

static void gpio_init(void) {
    gpio_export(GPIO_LED_GREEN);
    gpio_export(GPIO_LED_RED);
    gpio_write(GPIO_LED_GREEN, 0);
    gpio_write(GPIO_LED_RED, 0);
    gpio_no = 1;
}

static void led_granted(void) {
    if (!gpio_no) return;
    gpio_write(GPIO_LED_RED,   0);
    gpio_write(GPIO_LED_GREEN, 1);
}

static void led_denied(void) {
    if (!gpio_no) return;
    gpio_write(GPIO_LED_GREEN, 0);
    gpio_write(GPIO_LED_RED,   1);
}

static void led_off(void) {
    if (!gpio_no) return;
    gpio_write(GPIO_LED_GREEN, 0);
    gpio_write(GPIO_LED_RED,   0);
}

/* ---- bundle loading ---- */

static char *read_file(const char *path) {
    FILE *f = fopen(path, "r");
    if (!f) { perror(path); return NULL; }
    fseek(f, 0, SEEK_END);
    long len = ftell(f);
    rewind(f);
    char *buf = malloc(len + 1);
    if (!buf) { fclose(f); return NULL; }
    fread(buf, 1, len, f);
    buf[len] = '\0';
    fclose(f);
    return buf;
}

static char *read_stdin(void) {
    size_t cap = 4096, len = 0;
    char *buf = malloc(cap);
    if (!buf) return NULL;
    int c;
    while ((c = fgetc(stdin)) != EOF) {
        if (len + 1 >= cap) {
            cap *= 2;
            char *nb = realloc(buf, cap);
            if (!nb) { free(buf); return NULL; }
            buf = nb;
        }
        buf[len++] = (char)c;
    }
    buf[len] = '\0';
    return buf;
}

/* ---- main ---- */

int main(void) {
    const char *no_gpio = getenv("NO_GPIO");
    const char *bundle_file = getenv("BUNDLE_FILE");

    if (!no_gpio) {
        gpio_init();
    }

    printf("Ratify Physical Verifier  v%s\n", ratify_version());
    printf("Scope required: %s\n\n", SCOPE_PHYSICAL_ACCESS);

    char *bundle_json = NULL;
    if (bundle_file) {
        printf("Reading bundle from: %s\n", bundle_file);
        bundle_json = read_file(bundle_file);
    } else {
        printf("Reading bundle from stdin...\n");
        bundle_json = read_stdin();
    }

    if (!bundle_json || bundle_json[0] == '\0') {
        fprintf(stderr, "ERROR: empty or missing bundle\n");
        led_denied();
        return 1;
    }

    RatifyVerifyResult *result = NULL;
    char *err = NULL;
    RatifyStatus status = ratify_verify_bundle(
        bundle_json,
        SCOPE_PHYSICAL_ACCESS,
        0,          /* 0 = use system clock */
        &result,
        &err
    );

    free(bundle_json);

    if (status != RatifyOk) {
        fprintf(stderr, "ERROR: ratify_verify_bundle failed: %s\n",
                err ? err : "(unknown)");
        ratify_error_free(err);
        led_denied();
        return 1;
    }

    int valid = ratify_verify_result_is_valid(result);
    char *identity_status = ratify_verify_result_identity_status(result);
    char *agent_id        = ratify_verify_result_agent_id(result);
    char *human_id        = ratify_verify_result_human_id(result);
    char *error_reason    = ratify_verify_result_error_reason(result);

    if (valid) {
        printf("PHYSICAL_ACCESS GRANTED\n");
        printf("  agent_id  = %s\n", agent_id);
        printf("  human_id  = %s\n", human_id);
        printf("  status    = %s\n", identity_status);
        led_granted();
        sleep(5);   /* hold LED for 5 seconds */
        led_off();
    } else {
        printf("PHYSICAL_ACCESS DENIED\n");
        printf("  status = %s\n", identity_status);
        printf("  reason = %s\n", error_reason);
        led_denied();
        sleep(3);
        led_off();
    }

    ratify_string_free(identity_status);
    ratify_string_free(agent_id);
    ratify_string_free(human_id);
    ratify_string_free(error_reason);
    ratify_verify_result_free(result);

    return valid ? 0 : 1;
}
