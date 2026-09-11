import { Outlet } from 'react-router-dom';
import Box from '@mui/material/Box';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import LockOutlined from '@mui/icons-material/LockOutlined';
import { BrandMark } from '@/shared/components';
import heroImage from '@/assets/login_background.jpg';

const HERO_IMAGE_SRC: string = heroImage;

/**
 * Full-bleed split layout for unauthenticated pages (login, register,
 * forgot-password): a dark marketing panel on the left, the page's own
 * content (rendered via Outlet) centered on the right. Deliberately does
 * not use AppShell — there's no app nav to show before someone's
 * signed in.
 */
export function AuthLayout() {
  return (
    <Box sx={{ minHeight: '100vh', display: 'flex' }}>
      <Box
        sx={{
          display: { xs: 'none', md: 'flex' },
          width: '42%',
          flexShrink: 0,
          flexDirection: 'column',
          justifyContent: 'space-between',
          p: 6,
          color: 'common.white',
          backgroundColor: 'grey.900',
          backgroundImage: `linear-gradient(180deg, rgba(20,24,31,0.45) 0%, rgba(20,24,31,0.55) 40%, rgba(20,24,31,0.92) 100%), url(${HERO_IMAGE_SRC})`,
          backgroundSize: 'cover',
          backgroundPosition: 'center',
        }}
      >
        {/* Anchors the panel's top, which would otherwise be bare sky/photo
            with nothing to give it purpose. Hidden in the right-hand panel
            below on desktop (md+) to avoid showing the brand mark twice;
            it reappears there once this whole panel is hidden on mobile. */}
        <BrandMark light size={28} />

        <Typography variant="h2" sx={{ maxWidth: 440, fontWeight: 800, fontSize: '2.25rem', lineHeight: 1.25 }}>
          Every unit, lease, and dollar — in one place you can hand to an auditor.
        </Typography>
      </Box>

      <Box
        sx={{
          flex: 1,
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          p: 3,
          bgcolor: 'background.paper',
        }}
      >
        <Box sx={{ width: '100%', maxWidth: 400 }}>
          <Outlet />
        </Box>

        <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mt: 5 }}>
          <LockOutlined sx={{ fontSize: 16, color: 'text.disabled' }} />
          <Typography variant="caption" color="text.secondary">
            Your data is encrypted in transit and at rest.
          </Typography>
        </Stack>
      </Box>
    </Box>
  );
}
