#include <stdio.h>
#include <stdint.h>
#include <math.h>

#define PI 3.141592653589793
#define SOLAR_MASS (4.0 * PI * PI)
#define DAYS_PER_YEAR 365.24
#define N 5
#define STEPS 500000
#define DT 0.01

int main() {
    double x[N] = {0.0, 4.84143144246472090, 8.34336671824457987, 12.8943695621391310, 15.3796971148509165};
    double y[N] = {0.0, -1.16032004402742839, 4.12479856412430479, -15.1111514016986312, -25.9193146099879641};
    double z[N] = {0.0, -0.103622044471123109, -0.403523417114321381, -0.223307578892655734, 0.179258772950371181};

    double vx[N] = {0.0, 0.00166007664274403694*DAYS_PER_YEAR, -0.00276742510726862411*DAYS_PER_YEAR, 0.00296460137564761618*DAYS_PER_YEAR, 0.00268067772490389322*DAYS_PER_YEAR};
    double vy[N] = {0.0, 0.00769901118419740425*DAYS_PER_YEAR, 0.00499852801234917238*DAYS_PER_YEAR, 0.00237847173959480950*DAYS_PER_YEAR, 0.00162824170038242295*DAYS_PER_YEAR};
    double vz[N] = {0.0, -0.0000690460016972063023*DAYS_PER_YEAR, 0.0000230417297573763929*DAYS_PER_YEAR, -0.0000296589568540237556*DAYS_PER_YEAR, -0.0000951592254519715870*DAYS_PER_YEAR};

    double mass[N] = {SOLAR_MASS, 0.000954791938424326609*SOLAR_MASS, 0.000285885980666130812*SOLAR_MASS, 0.0000436624404335156298*SOLAR_MASS, 0.0000515138902046611451*SOLAR_MASS};

    /* Offset momentum */
    double px = 0, py = 0, pz = 0;
    for (int i = 0; i < N; i++) {
        px += vx[i] * mass[i];
        py += vy[i] * mass[i];
        pz += vz[i] * mass[i];
    }
    vx[0] = -px / SOLAR_MASS;
    vy[0] = -py / SOLAR_MASS;
    vz[0] = -pz / SOLAR_MASS;

    /* Advance */
    for (int step = 0; step < STEPS; step++) {
        for (int i = 0; i < N; i++) {
            for (int j = i + 1; j < N; j++) {
                double dx = x[i] - x[j];
                double dy = y[i] - y[j];
                double dz = z[i] - z[j];
                double dist2 = dx*dx + dy*dy + dz*dz;
                double dist = sqrt(dist2);
                double mag = DT / (dist2 * dist);
                vx[i] -= dx * mass[j] * mag;
                vy[i] -= dy * mass[j] * mag;
                vz[i] -= dz * mass[j] * mag;
                vx[j] += dx * mass[i] * mag;
                vy[j] += dy * mass[i] * mag;
                vz[j] += dz * mass[i] * mag;
            }
        }
        for (int i = 0; i < N; i++) {
            x[i] += DT * vx[i];
            y[i] += DT * vy[i];
            z[i] += DT * vz[i];
        }
    }

    /* Energy */
    double e = 0;
    for (int i = 0; i < N; i++) {
        e += 0.5 * mass[i] * (vx[i]*vx[i] + vy[i]*vy[i] + vz[i]*vz[i]);
        for (int j = i + 1; j < N; j++) {
            double dx = x[i] - x[j];
            double dy = y[i] - y[j];
            double dz = z[i] - z[j];
            double dist = sqrt(dx*dx + dy*dy + dz*dz);
            e -= mass[i] * mass[j] / dist;
        }
    }
    printf("%lld\n", (int64_t)(e * 1000000000.0));
    return 0;
}
