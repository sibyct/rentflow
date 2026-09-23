import { useMemo, useState, type ComponentType } from 'react';
import { useNavigate } from 'react-router-dom';
import Box from '@mui/material/Box';
import ButtonBase from '@mui/material/ButtonBase';
import InputAdornment from '@mui/material/InputAdornment';
import TextField from '@mui/material/TextField';
import Typography from '@mui/material/Typography';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import SearchOutlined from '@mui/icons-material/SearchOutlined';
import BusinessOutlined from '@mui/icons-material/BusinessOutlined';
import ShieldOutlined from '@mui/icons-material/ShieldOutlined';
import CreditCardOutlined from '@mui/icons-material/CreditCardOutlined';
import PeopleAltOutlined from '@mui/icons-material/PeopleAltOutlined';
import GroupsOutlined from '@mui/icons-material/GroupsOutlined';
import HomeWorkOutlined from '@mui/icons-material/HomeWorkOutlined';
import StorefrontOutlined from '@mui/icons-material/StorefrontOutlined';
import GavelOutlined from '@mui/icons-material/GavelOutlined';
import CalculateOutlined from '@mui/icons-material/CalculateOutlined';
import NotificationsOutlined from '@mui/icons-material/NotificationsOutlined';
import AccountBoxOutlined from '@mui/icons-material/AccountBoxOutlined';
import DescriptionOutlined from '@mui/icons-material/DescriptionOutlined';
import ExtensionOutlined from '@mui/icons-material/ExtensionOutlined';
import { tokens } from '@/app/tokens';
import { EmptyState } from '@/shared/components';

interface SettingsCardDef {
  key: string;
  title: string;
  description: string;
  icon: ComponentType<SvgIconProps>;
  /** Present only for the one section that's actually built. */
  path?: string;
}

interface SettingsGroupDef {
  label: string;
  cards: SettingsCardDef[];
}

const GROUPS: SettingsGroupDef[] = [
  {
    label: 'Organization',
    cards: [
      { key: 'company', title: 'Company profile', description: 'Legal name, address, logo and contacts.', icon: BusinessOutlined },
      { key: 'security', title: 'Security', description: 'Two-factor authentication and session policy.', icon: ShieldOutlined },
      { key: 'billing', title: 'Billing & plan', description: 'RentFlow subscription and invoices.', icon: CreditCardOutlined },
    ],
  },
  {
    label: 'People',
    cards: [
      { key: 'users', title: 'Users & roles', description: 'Staff, roles and property access.', icon: PeopleAltOutlined, path: '/settings/users-roles' },
      { key: 'owners', title: 'Owners', description: 'Owner records and portfolio reporting access.', icon: GroupsOutlined },
    ],
  },
  {
    label: 'Portfolio',
    cards: [
      { key: 'portfolio-defaults', title: 'Portfolio defaults', description: 'Default lease terms, units and property fields.', icon: HomeWorkOutlined },
      { key: 'vendor-defaults', title: 'Vendor categories & defaults', description: 'Trade categories, rate types and payment terms.', icon: StorefrontOutlined },
      { key: 'compliance', title: 'Compliance & legal', description: 'Required documents, insurance and licensing rules.', icon: GavelOutlined },
    ],
  },
  {
    label: 'Finance',
    cards: [
      { key: 'accounting', title: 'Accounting', description: 'Chart of accounts, late fees and payout schedule.', icon: CalculateOutlined },
    ],
  },
  {
    label: 'Residents & Comms',
    cards: [
      { key: 'notifications', title: 'Notifications', description: 'Email and SMS alerts for staff and residents.', icon: NotificationsOutlined },
      { key: 'portal', title: 'Tenant portal', description: 'Resident sign-in, payments and portal branding.', icon: AccountBoxOutlined },
      { key: 'templates', title: 'Document templates', description: 'Lease, notice and disclosure templates.', icon: DescriptionOutlined },
    ],
  },
  {
    label: 'Platform',
    cards: [
      { key: 'integrations', title: 'Integrations', description: 'Connected accounting, screening and calendar apps.', icon: ExtensionOutlined },
    ],
  },
];

/**
 * Settings home: a search box plus every settings section as a card,
 * grouped under headers. Only "Users & roles" is wired to a real screen
 * — everything else is a disabled placeholder, honestly labeled rather
 * than faked, matching this app's convention elsewhere (see the
 * Dashboard/Reports nav items).
 */
export function SettingsHomeScreen() {
  const navigate = useNavigate();
  const [search, setSearch] = useState('');

  const filteredGroups = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return GROUPS;
    return GROUPS.map((g) => ({
      ...g,
      cards: g.cards.filter((c) => c.title.toLowerCase().includes(q) || c.description.toLowerCase().includes(q)),
    })).filter((g) => g.cards.length > 0);
  }, [search]);

  return (
    <Box sx={{ maxWidth: 980 }}>
      <TextField
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Find a setting"
        sx={{ width: '100%', maxWidth: 420, mb: 4 }}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <SearchOutlined sx={{ fontSize: 18, color: tokens.slate[400] }} />
              </InputAdornment>
            ),
          },
        }}
      />

      {filteredGroups.length === 0 ? (
        <EmptyState icon={SearchOutlined} title="No settings match your search" description={`Nothing found for "${search}".`} />
      ) : (
        filteredGroups.map((group) => (
          <Box key={group.label} sx={{ mb: 4 }}>
            <Typography sx={{ fontSize: 11, fontWeight: 700, letterSpacing: '0.07em', textTransform: 'uppercase', color: tokens.slate[400], mb: 1.5 }}>
              {group.label}
            </Typography>
            <Box sx={{ display: 'grid', gap: 2, gridTemplateColumns: { xs: '1fr', sm: 'repeat(2, 1fr)', md: 'repeat(3, 1fr)' } }}>
              {group.cards.map((card) => {
                const Icon = card.icon;
                const enabled = Boolean(card.path);
                return (
                  <ButtonBase
                    key={card.key}
                    disabled={!enabled}
                    onClick={enabled ? () => navigate(card.path!) : undefined}
                    focusRipple
                    sx={{
                      textAlign: 'left',
                      alignItems: 'flex-start',
                      gap: 1.5,
                      p: 2.5,
                      bgcolor: 'common.white',
                      border: `1px solid ${tokens.slate[200]}`,
                      borderRadius: `${tokens.radiusCard}px`,
                      boxShadow: tokens.shadowCard,
                      opacity: enabled ? 1 : 0.55,
                      transition: 'box-shadow .15s, transform .15s, border-color .15s',
                      '&:hover': enabled
                        ? { boxShadow: tokens.shadowPopover, transform: 'translateY(-2px)', borderColor: tokens.azure[200] }
                        : undefined,
                    }}
                  >
                    <Box
                      sx={{
                        width: 38,
                        height: 38,
                        borderRadius: '9px',
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        flexShrink: 0,
                        bgcolor: enabled ? tokens.azure[50] : tokens.slate[100],
                        color: enabled ? tokens.azure[600] : tokens.slate[400],
                      }}
                    >
                      <Icon sx={{ fontSize: 20 }} />
                    </Box>
                    <Box>
                      <Typography sx={{ fontSize: 14.5, fontWeight: 700, color: enabled ? tokens.slate[900] : tokens.slate[600], mb: 0.4 }}>
                        {card.title}
                      </Typography>
                      <Typography sx={{ fontSize: 12.5, color: tokens.slate[enabled ? 500 : 400], lineHeight: 1.5 }}>
                        {card.description}
                      </Typography>
                    </Box>
                  </ButtonBase>
                );
              })}
            </Box>
          </Box>
        ))
      )}
    </Box>
  );
}
