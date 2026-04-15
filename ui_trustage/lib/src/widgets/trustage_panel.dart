import 'package:flutter/material.dart';

/// A rounded card container with optional eyebrow, title, and actions.
class TrustagePanel extends StatelessWidget {
  const TrustagePanel({
    super.key,
    this.eyebrow,
    this.title,
    this.subtitle,
    this.actions,
    required this.child,
  });

  final String? eyebrow;
  final String? title;
  final String? subtitle;
  final List<Widget>? actions;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);

    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(16),
        side: BorderSide(color: theme.colorScheme.outlineVariant),
      ),
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            if (eyebrow != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 4),
                child: Text(
                  eyebrow!.toUpperCase(),
                  style: TextStyle(
                    fontSize: 10,
                    fontWeight: FontWeight.w600,
                    letterSpacing: 1.5,
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
            if (title != null || actions != null)
              Padding(
                padding: const EdgeInsets.only(bottom: 12),
                child: Row(
                  children: [
                    if (title != null)
                      Expanded(
                        child: Column(
                          crossAxisAlignment: CrossAxisAlignment.start,
                          children: [
                            Text(title!, style: theme.textTheme.titleMedium),
                            if (subtitle != null)
                              Text(
                                subtitle!,
                                style: theme.textTheme.bodySmall?.copyWith(
                                  color: theme.colorScheme.onSurfaceVariant,
                                ),
                              ),
                          ],
                        ),
                      ),
                    if (actions != null) ...actions!,
                  ],
                ),
              ),
            child,
          ],
        ),
      ),
    );
  }
}
